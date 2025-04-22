# Contributing to Sync Manager

Thank you for your interest in contributing to Sync Manager! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md) to help us maintain a healthy and welcoming community.

## Getting Started

1. **Fork the repository**: Start by forking the repository to your GitHub account.

2. **Clone your fork**: Clone your fork to your local machine.
   ```bash
   git clone https://github.com/your-username/sync-manager.git
   cd sync-manager
   ```

3. **Set up the development environment**: Follow the setup instructions in the [README.md](README.md).

4. **Create a new branch**: Create a branch for your feature or bugfix.
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

1. **Write your code**: Make your changes following the coding standards detailed below.

2. **Write tests**: Add tests for your changes to ensure functionality and prevent regressions.

3. **Run tests locally**: Make sure all tests pass before submitting your changes.
   ```bash
   make test
   ```

4. **Format your code**: Ensure your code follows our style guidelines.
   ```bash
   make format
   ```

5. **Commit your changes**: Write clear, concise commit messages.
   ```bash
   git commit -m "Brief description of the change"
   ```

6. **Push to your fork**: Push your changes to your GitHub fork.
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Submit a pull request**: Create a pull request from your fork to the main repository.

## Coding Standards

### Go Code

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Include comments for functions and packages
- Aim for high test coverage (at least 80%)
- Use meaningful variable and function names

### Commit Messages

- Use the present tense ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line to 72 characters or fewer
- Reference issues and pull requests after the first line

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update the documentation if necessary
3. The PR should work for all supported platforms
4. PR needs at least one approval from a maintainer

## Reporting Bugs

- Use the GitHub issue tracker
- Include detailed steps to reproduce the bug
- Include any relevant logs or error messages
- Specify the version of the software you're using

## Feature Requests

Feature requests are welcome. Please provide:
- A clear description of the feature
- The motivation behind the feature
- Any alternative solutions you've considered

## Questions?

If you have any questions, feel free to open an issue or contact the maintainers.

Thank you for contributing to Sync Manager!