package lang

import (
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
	"github.com/manjunathrathod/claude-repo-factory/internal/spec"
)

// Java specific option keys.
const (
	OptJavaGroupID    = "java_group_id"
	OptJavaArtifactID = "java_artifact_id"
)

func init() { register(java) }

var java = &Definition{
	Meta: plugin.Descriptor{
		ID:          "java",
		DisplayName: "Java",
		Summary:     "Java 21 with Maven, JUnit 5, Spotless and SpotBugs",
		Aliases:     []string{"jvm"},
		Status:      plugin.StatusStable,
		ProjectTypes: []plugin.ProjectType{
			{ID: "library", DisplayName: "Library", Summary: "Published Maven artifact"},
			{ID: "service", DisplayName: "Spring Boot service", Summary: "HTTP service on Spring Boot"},
			{ID: "cli", DisplayName: "CLI", Summary: "Executable jar with a command line interface"},
		},
		DefaultProjectType: "service",
	},
	RequiredOptions: []OptionSpec{
		{
			Key:      OptJavaGroupID,
			Prompt:   "Maven groupId",
			Help:     "Reverse domain identifier, for example com.acme.widget.",
			Required: true,
			Default:  func(s spec.Spec) string { return "com.example." + pythonIdentifier(s.Name) },
		},
		{
			Key:      OptJavaArtifactID,
			Prompt:   "Maven artifactId",
			Help:     "Artifact name, for example widget-service.",
			Required: true,
			Default:  func(s spec.Spec) string { return strings.ToLower(s.Name) },
		},
	},
	Instruct: func(s spec.Spec) plugin.Instructions {
		return plugin.Instructions{
			Toolchain: strings.Join([]string{
				"- Java 21 (LTS), fixed by the maven-compiler-plugin release setting.",
				"- Maven is the build tool and the Maven Wrapper (`./mvnw`) is committed; always invoke the wrapper.",
				"- Spotless owns formatting, SpotBugs and Error Prone own static analysis.",
				"- Dependency versions are managed in one place through `dependencyManagement`; no version literals scattered across modules.",
			}, "\n"),
			Standards: strings.Join([]string{
				"- Prefer immutable types: records for data carriers, final fields, no setters.",
				"- Use constructor injection. Field injection and static singletons are not allowed.",
				"- Never return null from a public method; return `Optional` or an empty collection.",
				"- Catch specific exceptions and add context when rethrowing; never swallow an exception silently.",
				"- Public types and methods carry Javadoc that explains contracts, not restated syntax.",
			}, "\n"),
			Testing: strings.Join([]string{
				"- JUnit 5 with AssertJ assertions; Mockito only for genuine external collaborators.",
				"- Tests live in `src/test/java` mirroring the main package structure.",
				"- Use Testcontainers for anything that talks to a database or broker.",
				"- Every bug fix starts with a failing test that reproduces the bug.",
			}, "\n"),
			Security: strings.Join([]string{
				"- The OWASP dependency-check Maven goal must report no unsuppressed high severity findings.",
				"- Never build SQL by string concatenation; use parameterised queries.",
				"- Never deserialise untrusted input with Java native serialization.",
				"- Secrets come from the environment or a secret manager; never from a committed properties file.",
			}, "\n"),
			Architecture: strings.Join([]string{
				"- `src/main/java` for production code, `src/main/resources` for configuration.",
				"- Layer packages by feature first, then by technical role.",
				"- The domain layer has no framework imports; adapters depend on the domain, never the reverse.",
			}, "\n"),
			Commands: []plugin.Command{
				{Name: "build", Run: "./mvnw -B verify", Description: "Compile, test and run all quality gates"},
				{Name: "test", Run: "./mvnw -B test", Description: "Run the unit test suite"},
				{Name: "lint", Run: "./mvnw -B spotless:check spotbugs:check", Description: "Formatting and static analysis"},
				{Name: "format", Run: "./mvnw -B spotless:apply", Description: "Apply formatting"},
			},
		}
	},
}
