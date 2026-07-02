# Contributing to Build Accelerator

Thank you for your interest in contributing to Build Accelerator!

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please:

- Be respectful and constructive in discussions
- Follow our coding standards and style guidelines
- Test your changes thoroughly before submitting

## How to Contribute

### Reporting Issues

If you encounter bugs or have feature suggestions:

1. Check existing issues to avoid duplicates
2. Provide detailed reproduction steps
3. Include your environment (Windows version, Visual Studio version, MSBuild version)
4. Attach relevant logs from `%BUILDCACHE_HOME%\logs\`

### Pull Requests

We welcome contributions! Here's the process:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/improved-caching`)
3. Make your changes with clear commit messages
4. Add tests for new functionality
5. Update documentation as needed
6. Submit a pull request with a clear description

### Development Setup

To work on Build Accelerator:

```bash
# Clone your fork
git clone https://github.com/yourusername/build-accelerator.git
cd build-accelerator

# Install dashboard dependencies
cd dashboard
npm install

# Build the service (requires Visual Studio)
cd ../src
cl /c service.c
link /OUT:bcache.exe service.obj

# Build setup installer
cd ../tools
cl /O2 /Fe:setup.exe setup.c /link advapi32.lib shell32.lib
```

### Code Style

- **C/C++**: Follow Microsoft C++ coding conventions
- **JavaScript**: Use ES6+ features, 2-space indentation
- **Markdown**: Use GitHub-flavored markdown

### Testing

Before submitting:

- [ ] Code compiles without warnings
- [ ] Service starts and stops correctly
- [ ] Dashboard displays metrics
- [ ] Cache hit rate improves on repeated builds
- [ ] No memory leaks (verify with Application Verifier)

### Areas for Contribution

We'd especially appreciate help with:

- **Performance optimization**: Reducing cache lookup latency
- **Cross-compiler support**: Adding Clang/Intel C++ support
- **Remote cache backends**: S3, Azure Blob, GCS integration
- **Documentation**: Tutorials, troubleshooting guides
- **Build scripts**: CMake support, vcpkg integration

## Questions?

Open an issue with the `question` label, or join our Discord server.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
