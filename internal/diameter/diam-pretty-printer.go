package diameter

import (
	"fmt"
	"go-ocs/internal/logging"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
)

// PrintDiameterMessage prints a Diameter message as a simple AVP tree.
// Names are best-effort for now; we can add dictionary-based name lookup later.
func PrintDiameterMessage(m *diam.Message) {
	var prettyBuilder strings.Builder

	if !logging.IsDebug() {
		return
	}

	prettyBuilder.WriteString(fmt.Sprintf("Diameter Message: CommandCode: %d, appId: %d, flags: %d\n",
		m.Header.CommandCode,
		m.Header.ApplicationID,
		m.Header.CommandFlags,
	))

	dp := m.Dictionary()
	for _, avp := range m.AVP {
		printAVP(avp, 0, m.Header.ApplicationID, dp, &prettyBuilder)
	}

	//logging.Debug(prettyBuilder.String())

	// Keep it simple: start/end markers so humans can follow it even if a few lines interleave
	fmt.Fprintf(os.Stdout, "=== DIAMETER PRETTY PRINT START ===\n")
	fmt.Fprint(os.Stdout, prettyBuilder.String())
	fmt.Fprintf(os.Stdout, "=== DIAMETER PRETTY PRINT END ===\n")
}

func avpName(a *diam.AVP, appID uint32, dp *dict.Parser) string {
	if dp == nil {
		return "<unknown>"
	}
	// Try to find by (appID, vendorId, code) then fallback
	if def, err := dp.FindAVPWithVendor(appID, a.Code, a.VendorID); err == nil {
		return def.Name
	}
	if def, err := dp.FindAVPWithVendor(0, a.Code, a.VendorID); err == nil { // base app fallback
		return def.Name
	}
	return "<unknown>"
}

func printAVP(a *diam.AVP, indent int, appID uint32, dp *dict.Parser, prettyBuilder *strings.Builder) {
	prefix := strings.Repeat("  ", indent)

	// Best-effort name (we can add dictionary-based names later)
	name := avpName(a, appID, dp)

	// Grouped AVP: depending on how the AVP was decoded, the payload may be either
	// a fully-decoded *diam.GroupedAVP or a raw datatype.Grouped (byte array).
	if g, ok := a.Data.(*diam.GroupedAVP); ok {
		prettyBuilder.WriteString(formatLine(prefix, a.Code, name, "<Grouped>"))

		for _, child := range g.AVP {
			printAVP(child, indent+1, appID, dp, prettyBuilder)
		}
		return
	}
	if raw, ok := a.Data.(datatype.Grouped); ok {
		g, err := diam.DecodeGrouped(raw, appID, dp)
		if err != nil {
			prettyBuilder.WriteString(fmt.Sprintf("%s%d: %-30s <Grouped decode error: %v>\n", prefix, a.Code, name, err))
			return
		}

		prettyBuilder.WriteString(formatLine(prefix, a.Code, name, "<Grouped>"))
		for _, child := range g.AVP {
			printAVP(child, indent+1, appID, dp, prettyBuilder)
		}
		return
	}

	prettyBuilder.WriteString(formatLine(prefix, a.Code, name, formatAVPValue(a.Data)))
}

func formatLine(prefix string, code uint32, name string, value string) string {
	var avpLine strings.Builder

	avpLine.WriteString(fmt.Sprintf("%s%d: %s", prefix, code, name))
	for avpLine.Len() < 40 {
		if avpLine.Len()%2 == 0 {
			avpLine.WriteString(".")
		} else {
			avpLine.WriteString(" ")
		}
	}
	avpLine.WriteString(value)
	avpLine.WriteString("\n")

	return avpLine.String()
}

func formatAVPValue(v datatype.Type) string {
	switch t := v.(type) {
	case datatype.UTF8String:
		return string(t)

	case datatype.OctetString:
		if utf8.Valid([]byte(t)) {
			return string(t)
		}
		return fmt.Sprintf("0x%x", []byte(t))

	case datatype.DiameterIdentity:
		return string(t)

	case datatype.Enumerated:
		return fmt.Sprintf("%d", uint32(t))

	case datatype.Unsigned32:
		return fmt.Sprintf("%d", uint32(t))

	case datatype.Integer32:
		return fmt.Sprintf("%d", int32(t))

	case datatype.Unsigned64:
		return fmt.Sprintf("%d", uint64(t))

	case datatype.Time:
		return time.Time(t).Format(time.RFC1123Z)

	default:
		return fmt.Sprintf("%v", t)
	}
}
