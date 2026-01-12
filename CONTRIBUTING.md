# Contributing to S3 Resource Operator

Thank you for your interest in contributing to the S3 Resource Operator! We welcome contributions from the community.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

If you find a bug, please create an issue on GitHub with:
- A clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (Kubernetes version, S3 backend, etc.)

### Suggesting Features

Feature requests are welcome! Please create an issue describing:
- The problem you're trying to solve
- Your proposed solution
- Any alternatives you've considered

### Pull Requests

1. **Fork the repository** and create a branch from `develop`
2. **Make your changes** following the code style guidelines
3. **Add tests** for any new functionality
4. **Update documentation** if needed
5. **Ensure tests pass**: `pytest -v`
6. **Sign your commits** (see Developer Certificate of Origin below)
7. **Submit a pull request** to the `develop` branch

## Developer Certificate of Origin (DCO)

This project requires all commits to be signed off to certify that you have the right to submit the code under the project's license. This is done by adding a `Signed-off-by` line to your commit messages.

### How to Sign Your Commits

**Automatically sign all commits:**
```bash
git config --global format.signoff true
```

**Or sign individual commits:**
```bash
git commit -s -m "feat: your commit message"
```

This adds a line like:
```
Signed-off-by: Your Name <your.email@example.com>
```

**To fix commits that are missing sign-off:**
```bash
# For the last commit
git commit --amend --signoff --no-edit

# Then force push (if already pushed)
git push --force-with-lease

### Automated commits from CI

Commits created by GitHub Actions (for example the `github-actions[bot]` user) are signed automatically by repository workflows where configured. For example, the `sync` workflow amends merge commits with a `Signed-off-by` trailer so the DCO check passes.

If you need to exclude or annotate automated commits for any reason:
- Use `[skip ci]` in the commit message to avoid CI runs when appropriate.
- Prefer adding a `Signed-off-by` trailer in the workflow (recommended) instead of disabling DCO checks.
- Some organizations use a custom DCO exemption process; contact maintainers if you need an exception.
```

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/s3-resource-operator.git
cd s3-resource-operator

# Install dependencies
pip install -r requirements.txt
pip install -r requirements-test.txt

# Configure git commit message template (optional but recommended)
git config commit.template .gitmessage

# Run tests
pytest -v

# Run linting (optional)
helm lint ./helm
```

### Code Style

- Follow PEP 8 for Python code
- Use meaningful variable and function names
- Add docstrings to classes and functions
- Keep functions focused and testable

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for bucket versioning
fix: correct owner change logic
docs: update installation instructions
test: add tests for graceful shutdown
chore: update dependencies
```

**Tip:** Configure the commit message template to get helpful reminders:
```bash
git config commit.template .gitmessage
```

This will pre-populate your commit messages with the proper format and guidelines.

### Testing

- Write unit tests for new functionality
- Ensure all tests pass before submitting PR
- Test coverage should not decrease

### Documentation

- Update README.md for user-facing changes
- Update code comments and docstrings
- Add examples for new features

## Release Process

This project uses semantic-release for automated releases:
1. PRs are merged to `develop` branch
2. When ready, `develop` is merged to `main`
3. Semantic-release automatically creates version tags
4. Docker images and Helm charts are published to GHCR

## Questions?

Feel free to open an issue for any questions about contributing!
