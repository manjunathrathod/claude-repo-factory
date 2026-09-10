package lang

import (
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

// OptPackageName is the npm package name for Node.js repositories.
const OptPackageName = "package_name"

// Node.js package managers. Package manager identifiers are ecosystem
// knowledge, so they live beside the plugin that owns them, never in the core.
const (
	PMNpm  config.PackageManager = "npm"
	PMPnpm config.PackageManager = "pnpm"
	PMYarn config.PackageManager = "yarn"
)

func init() { register(nodejs) }

var nodejs = &Definition{
	Meta: plugin.Descriptor{
		ID:          "nodejs",
		DisplayName: "Node.js / TypeScript",
		Summary:     "TypeScript on Node.js with ESLint, Prettier and Vitest",
		Aliases:     []string{"node", "ts", "typescript", "javascript", "js"},
		Status:      plugin.StatusStable,
		ProjectTypes: []plugin.ProjectType{
			{ID: config.ProjectTypeAPI, DisplayName: "API", Summary: "HTTP service on Node.js"},
			{ID: config.ProjectTypeCLI, DisplayName: "CLI", Summary: "Command line tool published to npm"},
			{ID: config.ProjectTypeLibrary, DisplayName: "Library", Summary: "Reusable package with a public API"},
			{ID: config.ProjectTypeWorker, DisplayName: "Worker", Summary: "Queue consumer or scheduled job"},
		},
		DefaultProjectType:    config.ProjectTypeLibrary,
		PackageManagers:       []config.PackageManager{PMNpm, PMPnpm, PMYarn},
		DefaultPackageManager: PMNpm,
	},
	RequiredOptions: []OptionSpec{{
		Key:      OptPackageName,
		Prompt:   "npm package name",
		Help:     "Published package name, for example @acme/widget or widget.",
		Required: true,
		Default:  func(c config.ProjectConfig) string { return strings.ToLower(c.ProjectName) },
	}},
	Instruct: func(c config.ProjectConfig) plugin.Instructions {
		return plugin.Instructions{
			Toolchain: strings.Join([]string{
				"- Node.js 22 LTS, pinned in `.nvmrc` and in the `engines` field of `package.json`.",
				"- `npm` is the package manager. `package-lock.json` is committed and must be updated in the same commit as `package.json`.",
				"- TypeScript is compiled in `strict` mode. `any` is not allowed; use `unknown` plus a narrowing check.",
				"- ESLint and Prettier own formatting. Never hand-format code that a tool can format.",
			}, "\n"),
			Standards: strings.Join([]string{
				"- ES modules only (`\"type\": \"module\"`). No `require`.",
				"- Every exported symbol has an explicit return type and a doc comment.",
				"- Validate all external input at the boundary (zod or an equivalent schema), then trust it inside.",
				"- Prefer `async`/`await` over raw promise chains; never leave a floating promise unawaited.",
				"- Errors are thrown as `Error` subclasses carrying a stable `code`, never as strings.",
			}, "\n"),
			Testing: strings.Join([]string{
				"- Vitest is the test runner. Unit tests live beside the source as `*.test.ts`; integration tests live in `tests/`.",
				"- Every bug fix starts with a failing test that reproduces the bug.",
				"- Do not mock what you own; mock only the network, the clock and the filesystem.",
				"- Coverage must not fall below the threshold configured in `vitest.config.ts`.",
			}, "\n"),
			Security: strings.Join([]string{
				"- `npm audit --omit=dev` must report no high or critical advisories before a release.",
				"- Never interpolate user input into `child_process` calls; use the array form of `execFile`.",
				"- Secrets come from the environment and are read once at start-up; never log them.",
				"- Pin GitHub Actions and Docker base images by digest, not by floating tag.",
			}, "\n"),
			Architecture: strings.Join([]string{
				"- `src/` holds the implementation, `src/index.ts` is the only public entry point.",
				"- `tests/` holds integration and end-to-end tests.",
				"- `dist/` is build output and is never committed.",
			}, "\n"),
			Commands: []plugin.Command{
				{Name: "install", Run: "npm ci", Description: "Install dependencies from the lockfile"},
				{Name: "build", Run: "npm run build", Description: "Type-check and emit to dist/"},
				{Name: "test", Run: "npm test", Description: "Run the full test suite"},
				{Name: "lint", Run: "npm run lint", Description: "ESLint with the project ruleset"},
				{Name: "format", Run: "npm run format", Description: "Apply Prettier formatting"},
			},
		}
	},
}
