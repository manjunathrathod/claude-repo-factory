# Security policy

## Supported versions

`claude-repo-factory` is pre-1.0. Only the latest release receives security
fixes.

## Reporting a vulnerability

Do not open a public issue.

Report privately to the maintainers with:

- a description of the problem and the class of vulnerability;
- the affected version (`claude-repo-factory version`);
- a minimal reproduction.

Describe the class of problem rather than publishing a working exploit. You
will get an acknowledgement, and a fix or an explanation of why the behaviour
is intended.

## Scope

This tool runs locally with filesystem write access, shells out to `git`, and
generates content that lands in other repositories. Both the tool and its
output are in scope — in particular command injection, path traversal outside
the target directory, unsafe file permissions, secrets in generated content,
and insecure generated CI.

The threat model and the rules that follow from it are in
[docs/security.md](docs/security.md).
