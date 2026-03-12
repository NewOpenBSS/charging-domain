package main

import (
	"context"
	"errors"
	"fmt"
	"go-ocs/internal/diameter"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
	"github.com/fiorix/go-diameter/v4/diam/sm"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// StartDRAServer initializes and starts the Diameter server using the provided configuration.
// Returns a stop function to gracefully shut down the server and an error if initialization fails.
func StartDRAServer(context *AppContext) (stop func(), err error) {
	logging.Info("Starting Diameter server")

	// get diameter settings
	d := context.Config.Diameter
	settings := toSMSettings(d)

	// Create the state machine (mux)
	mux := sm.New(settings)
	mux.Handle("CCR", handleCCR(*context))
	mux.Handle("DPR", handleDPR())
	mux.HandleFunc("ALL", handleALL) // Catch all.

	t := d.Transport

	// 1) Bind ALL listeners synchronously
	listeners := make([]net.Listener, 0, len(t.BindAddresses))
	for _, a := range t.BindAddresses {
		ln, e := diam.Listen(t.Network, a) // binds socket
		if e != nil {
			for _, l := range listeners {
				_ = l.Close()
			}
			return nil, fmt.Errorf("failed to bind diameter on %s: %w", a, e)
		}
		listeners = append(listeners, ln)
		logging.Info("DiameterConfig bounded",
			"addr", a,
			"network", t.Network)
	}

	// A single place to close everything.
	stopOnce := sync.Once{}
	stop = func() {
		stopOnce.Do(func() {
			for _, l := range listeners {
				_ = l.Close()
			}
		})
	}

	errCh := make(chan error, len(t.BindAddresses))
	for i, ln := range listeners {
		addr := t.BindAddresses[i]
		srv := &diam.Server{
			Network: t.Network,
			Addr:    addr,
			Handler: mux,          // your sm mux
			Dict:    dict.Default, // e.g. dict.Default or your loaded parser
		}
		go func(a string, l net.Listener) {
			logging.Info("DiameterConfig serving", "addr", a)
			if e := srv.Serve(l); e != nil {
				// On ANY serve failure: stop everything else and report upward
				stop()
				select {
				case errCh <- fmt.Errorf("diameter server on %s stopped: %w", a, e):
				default:
				}
			}
		}(addr, ln)
	}

	go startSMErrorLogger(mux)

	return stop, nil
}

// startSMErrorLogger listens for error reports from the state machine and logs detailed error information for debugging and monitoring purposes.
func startSMErrorLogger(mux *sm.StateMachine) {
	for er := range mux.ErrorReports() {
		if er.Message != nil {
			logging.Error("sm error.",
				"from", er.Conn.RemoteAddr(),
				"error", er.Error,
				"code", er.Message.Header.CommandCode,
				"applId", er.Message.Header.ApplicationID,
				"flags", er.Message.Header.CommandFlags,
				"msg", er.Message,
			)
		} else {
			logging.Error("sm decoding failed before header/avps were available)",
				"from", er.Conn.RemoteAddr(),
				"error", er.Error,
			)
		}
	}
}

// toSMSettings converts a DiameterFile configuration into an *sm.Settings object for initializing the state machine.
func toSMSettings(d DiameterFile) *sm.Settings {
	return &sm.Settings{
		OriginHost:       datatype.DiameterIdentity(d.LocalPeer.Host),
		OriginRealm:      datatype.DiameterIdentity(d.LocalPeer.Realm),
		VendorID:         datatype.Unsigned32(d.LocalPeer.VendorID),
		ProductName:      datatype.UTF8String(d.LocalPeer.ProductName),
		OriginStateID:    datatype.Unsigned32(d.LocalPeer.OriginStateID),
		FirmwareRevision: datatype.Unsigned32(d.LocalPeer.FirmwareRevision),
	}
}

// handleALL handles all incoming Diameter messages that are not explicitly matched by other handlers.
// Logs the message details and responds with UnableToComply.
func handleALL(c diam.Conn, m *diam.Message) {
	logging.Debug("Received unknown msg",
		"code", m.Header.CommandCode,
		"applId", m.Header.ApplicationID,
		"flags", m.Header.CommandFlags,
		"msg", m,
	)

	response := createAnswer(m, diam.UnableToComply)
	if _, err := response.WriteTo(c); err != nil {
		logging.Error("failed to send answer", "error", err)
	}
}

// createAnswer generates a Diameter answer message by copying specific AVPs from the request and setting the result code.
func createAnswer(m *diam.Message, resultCode uint32) *diam.Message {
	answer := m.Answer(resultCode)

	if a, err := m.FindAVP(avp.SessionID, 0); err == nil {
		answer.NewAVP(avp.SessionID, avp.Mbit, 0, a.Data)
	}

	if a, err := m.FindAVP(avp.VendorSpecificApplicationID, 0); err == nil {
		answer.NewAVP(avp.VendorSpecificApplicationID, avp.Mbit, 0, a.Data)
	}

	if a, err := m.FindAVP(avp.AcctApplicationID, 0); err == nil {
		answer.NewAVP(avp.AcctApplicationID, avp.Mbit, 0, a.Data)
	}

	if a, err := m.FindAVP(avp.AuthApplicationID, 0); err == nil {
		answer.NewAVP(avp.AuthApplicationID, avp.Mbit, 0, a.Data)
	}

	if a, err := m.FindAVP(avp.ProxyInfo, 0); err == nil {
		answer.NewAVP(avp.ProxyInfo, avp.Mbit, 0, a.Data)
	}

	// Copy route information
	if a, err := m.FindAVP(avp.OriginHost, 0); err == nil {
		answer.NewAVP(avp.DestinationHost, avp.Mbit, 0, a.Data)
	}

	if a, err := m.FindAVP(avp.OriginRealm, 0); err == nil {
		answer.NewAVP(avp.DestinationRealm, avp.Mbit, 0, a.Data)
	}

	return answer
}

func handleDPR() diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		// DPA
		a := m.Answer(diam.Success)

		if _, err := a.WriteTo(c); err != nil {
			logging.Error("failed to send DPA", "error", err)
		}

		logging.Debug("Disconnect Peer Request received", "addr", c.RemoteAddr())
	}
}

// handleCCR handles a Credit Control Request (CCR) message and communicates with the OCS to process charging operations.
func handleCCR(ac AppContext) diam.HandlerFunc {
	return func(c diam.Conn, m *diam.Message) {
		startTime := time.Now()
		ac.Metrics.IncRate()

		logging.Debug("Received CCR message", "addr", c.RemoteAddr())

		diameter.PrintDiameterMessage(m)

		request, err := nchf.AvpToNchfRequest(m)
		if err != nil {
			sendErrorResponse(ac, c, m, diam.UnableToComply, err.Error())
			return
		}

		// Nationalise the MSISDN
		msisdn := *request.SubscriberIdentifier
		if len(msisdn) > 0 && msisdn[0] != '0' {
			nationalDialCode := ac.Config.ChargingDRA.NationalDialCode

			if strings.HasPrefix(msisdn, nationalDialCode) {
				msisdn = "0" + msisdn[len(nationalDialCode):]
			} else {
				msisdn = "0" + msisdn
			}
		}

		subcriber, err := ac.Store.Q.FindSubscriberWithWholesalerByMSISDN(context.Background(), msisdn)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				sendErrorResponse(ac, c, m, diam.UnableToComply, "Subscriber not found")
			} else {
				sendErrorResponse(ac, c, m, diam.UnableToComply, err.Error())
			}
			return
		}

		// Get the per-wholesaler limiter (keyed by wholesaler_id)
		whID, err := uuid.FromBytes(subcriber.WholesalerID.Bytes[:])
		if err != nil {
			sendErrorResponse(ac, c, m, diam.UnableToComply, err.Error())
			return
		}
		rpm, err := subcriber.WholesalerRatelimit.Float64Value()
		if err != nil {
			sendErrorResponse(ac, c, m, diam.UnableToComply, err.Error())
			return
		}

		lim := ac.Limiter.Get(whID, rpm.Float64, 50)

		// Enforce (reject fast is usually best for Diameter)
		if !lim.Allow() {
			sendErrorResponse(ac, c, m, diam.TooBusy, "Rate limit exceeded")
			return
		}

		// Create request-scoped context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		url := subcriber.WholesalerNchfurl + "/nchf-charging"
		response, err := ac.Ocs.Send(ctx, url, request)
		if err != nil {
			logging.Error("Failed to send request to OCS", "error", err)
			sendErrorResponse(ac, c, m, diam.UnableToComply, "Failed to send request to OCS")
			return
		}

		answer := createAnswer(m, diam.Success)

		resultCode, err := nchf.NhcfToAvpResponse(answer, response)
		if err != nil {
			logging.Error("Failed to create answer", "error", err)
			sendErrorResponse(ac, c, m, diam.UnableToComply, err.Error())
			return
		}

		if response.Runtime == nil {
			response.Runtime = new(int64)
			*response.Runtime = 0
		}

		duration := time.Since(startTime)
		runtime := time.Duration(*response.Runtime) * time.Millisecond
		ac.Metrics.ObserveRuntimeOverhead(response.GetRequestTypeString(), duration, runtime)

		logging.Info("Reply received",
			"sessionId", request.ChargingId,
			"subscriberId", request.SubscriberIdentifier,
			"resultCode", resultCode,
			"runtime", runtime.Milliseconds(),
			"duration", duration.Milliseconds(),
		)

		diameter.PrintDiameterMessage(answer)

		if _, err := answer.WriteTo(c); err != nil {
			logging.Error("Failed to send answer", "error", err)
		}
	}
}

// sendErrorResponse sends a Diameter error response with the specified result code and error message to the requesting connection.
func sendErrorResponse(ctx AppContext, c diam.Conn, m *diam.Message, resultCode uint32, errorText string) {
	ctx.Metrics.IncErrorRate()

	response := createAnswer(m, resultCode)

	a, err := m.FindAVP(avp.CCRequestType, 0)
	if err != nil {
		logging.Error("failed to find RequestType AVP", "error", err)
		return
	}

	if _, err = response.NewAVP(avp.CCRequestType, avp.Mbit, 0, a.Data); err != nil {
		logging.Error("failed to add RequestType AVP", "error", err)
	}

	if _, err = response.NewAVP(avp.ErrorMessage, avp.Mbit, 0, datatype.UTF8String(errorText)); err != nil {
		logging.Error("failed to add ErrorMessage AVP", "error", err)
		return
	}

	if _, err := response.WriteTo(c); err != nil {
		logging.Error("failed to send answer", "error", err)
	}

	return
}
