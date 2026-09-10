package lang

import (
	"strings"

	"github.com/manjunathrathod/claude-repo-factory/internal/config"
	"github.com/manjunathrathod/claude-repo-factory/internal/plugin"
)

// OptPythonPackage is the importable package name for Python repositories.
const OptPythonPackage = "python_package"

// Python package managers.
const (
	PMPip    config.PackageManager = "pip"
	PMUv     config.PackageManager = "uv"
	PMPoetry config.PackageManager = "poetry"
)

func init() { register(python) }

var python = &Definition{
	Meta: plugin.Descriptor{
		ID:          "python",
		DisplayName: "Python",
		Summary:     "Python 3.12 with uv, ruff, mypy and pytest",
		Aliases:     []string{"py", "python3"},
		Status:      plugin.StatusStable,
		ProjectTypes: []plugin.ProjectType{
			{ID: config.ProjectTypeAPI, DisplayName: "API", Summary: "FastAPI service"},
			{ID: config.ProjectTypeCLI, DisplayName: "CLI", Summary: "Command line tool with a console entry point"},
			{ID: config.ProjectTypeLibrary, DisplayName: "Library", Summary: "Distributable package"},
			{ID: config.ProjectTypeWorker, DisplayName: "Worker", Summary: "Background processor or data pipeline"},
		},
		DefaultProjectType:    config.ProjectTypeLibrary,
		PackageManagers:       []config.PackageManager{PMUv, PMPip, PMPoetry},
		DefaultPackageManager: PMUv,
	},
	RequiredOptions: []OptionSpec{{
		Key:      OptPythonPackage,
		Prompt:   "Python package name",
		Help:     "Importable package name; lowercase with underscores, for example my_widget.",
		Required: true,
		Default:  func(c config.ProjectConfig) string { return pythonIdentifier(c.ProjectName) },
	}},
	Instruct: func(c config.ProjectConfig) plugin.Instructions {
		return plugin.Instructions{
			Toolchain: strings.Join([]string{
				"- Python 3.12, declared in `pyproject.toml` under `requires-python`.",
				"- `uv` manages the virtual environment and the lockfile. `uv.lock` is committed.",
				"- `ruff` handles both linting and formatting. `mypy --strict` type-checks the package.",
				"- All configuration lives in `pyproject.toml`. Do not add setup.py or setup.cfg.",
			}, "\n"),
			Standards: strings.Join([]string{
				"- Type annotations are mandatory on every function signature, including tests.",
				"- Public modules, classes and functions carry docstrings in Google style.",
				"- Prefer dataclasses or pydantic models over dictionaries for structured data.",
				"- Catch specific exceptions. A bare `except:` or a bare `except Exception:` without a re-raise is a defect.",
				"- No mutable default arguments; no module-level side effects on import.",
			}, "\n"),
			Testing: strings.Join([]string{
				"- `pytest` is the test runner; tests live in `tests/` mirroring the package layout.",
				"- Use fixtures over setup methods, and `pytest.mark.parametrize` over copy-pasted cases.",
				"- Every bug fix starts with a failing test that reproduces the bug.",
				"- Coverage is measured with `pytest --cov` and must not regress.",
			}, "\n"),
			Security: strings.Join([]string{
				"- `pip-audit` (or `uv pip audit`) must report no known vulnerabilities before a release.",
				"- Never call `eval`, `exec`, `pickle.loads` or `yaml.load` on untrusted input; use `yaml.safe_load`.",
				"- Use `subprocess.run` with a list argument and `shell=False`.",
				"- Secrets come from the environment; never commit a `.env` file.",
			}, "\n"),
			Architecture: strings.Join([]string{
				"- `src/` layout: the package lives in `src/<package>/` so tests run against the installed distribution.",
				"- `tests/` mirrors the package structure one-to-one.",
				"- Business logic never imports the CLI or web layer; dependencies point inward.",
			}, "\n"),
			Commands: []plugin.Command{
				{Name: "install", Run: "uv sync --all-extras --dev", Description: "Create the environment from the lockfile"},
				{Name: "test", Run: "uv run pytest", Description: "Run the full test suite"},
				{Name: "lint", Run: "uv run ruff check .", Description: "Lint the project"},
				{Name: "format", Run: "uv run ruff format .", Description: "Apply formatting"},
				{Name: "typecheck", Run: "uv run mypy src", Description: "Strict static type check"},
			},
		}
	},
}

// pythonIdentifier converts a repository name into a valid, importable
// package name.
func pythonIdentifier(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" || (out[0] >= '0' && out[0] <= '9') {
		out = "pkg_" + out
	}
	return out
}
