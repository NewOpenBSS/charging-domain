# Java — Language and Framework Standards

Apply these rules when working in any Java package in this repository.

> **Note:** This file is a starter template. Expand it when the first Java project
> is brought into the agentic development process. Add project-specific frameworks,
> build tools, and conventions as they become known.

---

## Build Verification

After any change to Java source or dependencies — run the appropriate command
for the build tool in use:

**Maven:**
```bash
mvn clean verify
```

**Gradle:**
```bash
./gradlew clean build
```

Never claim an implementation is complete without the build passing cleanly.
Report exact output on any failure — diagnose before retrying.

---

## Code Standards

**Null safety**
Avoid returning `null` from methods where possible. Use `Optional<T>` for values
that may be absent. Always check for null before dereferencing where nulls are
possible.

**Immutability**
Prefer immutable objects. Use `final` fields where possible. Avoid mutable shared
state.

**Exception handling**
Use checked exceptions for recoverable conditions, unchecked (RuntimeException)
for programming errors. Never swallow exceptions silently — log or rethrow.
Never use exceptions for control flow.

**Naming conventions**
Follow standard Java naming: `PascalCase` for classes, `camelCase` for methods
and variables, `UPPER_SNAKE_CASE` for constants, `lowercase.with.dots` for packages.


---

## Testing

**Frameworks:**
- JUnit 5 (`@Test`, `@BeforeEach`, `@AfterEach`, `@ParameterizedTest`)
- Mockito for mocking dependencies
- AssertJ for fluent assertions (preferred over plain JUnit assertions)

**Commands:**
```bash
mvn test                          # Maven
./gradlew test                    # Gradle
./gradlew test --tests "ClassName" # specific test class
```

**Requirements:**
- Every class with business logic must have a corresponding test class
- Test class naming: `ClassNameTest`
- Use `@ParameterizedTest` with `@MethodSource` or `@CsvSource` for
  multiple input/output combinations — equivalent to Go's table-driven tests
- Mocks must be used for all external dependencies (database, HTTP, messaging)
- Tests must not require running external services

**Test naming:**
Method names should describe the scenario:
`methodName_givenCondition_expectedBehaviour`
e.g. `debitQuota_givenInsufficientBalance_throwsOutOfFundsException`

---

## Dependency Management

**Maven:** Add dependencies to `pom.xml`. Verify versions on
[Maven Central](https://search.maven.org) before adding.

**Gradle:** Add dependencies to `build.gradle` or `build.gradle.kts`.

- Prefer dependencies already used in the project
- Never add a dependency without confirming it exists at the specified version
- Scope dependencies correctly: `test` scope for test-only libraries

---

## Architecture Boundaries

> Expand this section when the specific architecture of the Java project
> is understood. Common patterns to document here:
> - Package structure conventions
> - Layer boundaries (controller / service / repository)
> - Framework-specific rules (Spring Boot, Quarkus, Micronaut, etc.)
> - ORM conventions (JPA, Hibernate, Panache)
> - Configuration approach (application.properties, application.yaml, etc.)

---

## Documentation

- All public classes and methods must have Javadoc
- `@param`, `@return`, and `@throws` tags required on public methods
- Comments must describe intent and non-obvious behaviour — not restate the code

