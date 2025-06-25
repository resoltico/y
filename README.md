# Otsu Obliterator

A high-performance image processing application implementing advanced Otsu thresholding algorithms with a modern GUI.

## Features

- **2D Otsu Thresholding**: Advanced histogram-based segmentation with neighborhood analysis
- **Iterative Triclass**: Multi-pass thresholding with convergence detection
- **Real-time Preview**: Interactive parameter adjustment with instant feedback
- **Cross-platform GUI**: Native desktop interface using Fyne
- **Memory Management**: Thread-safe OpenCV Mat handling with pooling
- **Performance Monitoring**: Built-in profiling and debug capabilities

## Quick Start

```bash
# Clone and build
git clone <repository-url>
cd otsu-obliterator
chmod +x build.sh

# Install dependencies and run
make deps
make run
```

## Requirements

- **Go 1.24+**
- **OpenCV 4.11.0+** (for computer vision operations)
- **pkg-config** (for OpenCV detection)

### Platform-specific Setup

**macOS:**
```bash
brew install opencv pkg-config
```

**Ubuntu/Debian:**
```bash
sudo apt install libopencv-dev pkg-config
```

**Windows:**
Install OpenCV and ensure pkg-config is available, or use pre-built binaries.

## Usage

### Basic Operations

```bash
# Build and run application
make run

# Build with profiling support
make build-profile

# Run with debugging
make debug-safe    # Safe debugging (no pixel analysis)
make debug-all     # Full debugging output
```

### Algorithm Parameters

**2D Otsu:**
- Window size (3-21, odd numbers)
- Histogram bins (16-256)
- Neighborhood metrics (mean, median, gaussian)
- Pixel weight factor (0.0-1.0)

**Iterative Triclass:**
- Threshold methods (otsu, mean, median)
- Convergence epsilon (0.1-10.0)
- Maximum iterations (1-20)
- Gap factor (0.0-1.0)

### Supported Formats

**Input:** JPEG, PNG, TIFF, BMP, GIF, WebP
**Output:** JPEG, PNG (with quality preservation)

## Development

### Building

```bash
./build.sh help                    # Show all options
./build.sh build                   # Standard build
./build.sh build profile           # With profiling
./build.sh build windows           # Cross-compile for Windows
```

### Testing

```bash
make test                          # Run test suite
./build.sh test                    # Alternative test runner
```

### Debugging

```bash
./build.sh debug format            # Image format debugging
./build.sh debug algorithms        # Algorithm execution tracing
./build.sh debug memory            # Memory usage monitoring
```

### Cross-compilation

```bash
make build-windows                 # Windows executable
make build-macos                   # macOS Intel binary
make build-macos-arm64             # macOS Apple Silicon binary
make build-linux                   # Linux binary
```

## Architecture

```
cmd/otsu-obliterator/      # Application entry point
internal/
├── algorithms/            # Processing algorithms
│   ├── otsu2d/           # 2D Otsu implementation
│   └── triclass/         # Iterative triclass algorithm
├── app/                  # Application coordination
├── gui/                  # Fyne-based interface
├── opencv/               # OpenCV integration
│   ├── safe/            # Thread-safe Mat wrappers
│   ├── bridge/          # Go/OpenCV conversions
│   └── memory/          # Memory management
└── pipeline/             # Image processing pipeline
```

## Performance

- **Memory pooling** reduces allocation overhead
- **Concurrent processing** utilizes multiple CPU cores
- **SIMD optimizations** through OpenCV acceleration
- **Progressive rendering** for real-time preview

## Profiling

Enable profiling server:
```bash
make build-profile
./build/otsu-obliterator &
```

Access profiling data:
- Memory: http://localhost:6060/debug/pprof/heap
- CPU: http://localhost:6060/debug/pprof/profile
- Goroutines: http://localhost:6060/debug/pprof/goroutine

## License

[Specify your license here]

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/algorithm-improvement`)
3. Commit changes (`git commit -am 'Add histogram optimization'`)
4. Push branch (`git push origin feature/algorithm-improvement`)
5. Open Pull Request

## Troubleshooting

**Build fails with OpenCV errors:**
- Verify OpenCV installation: `pkg-config --cflags opencv4`
- Check Go version: `go version`
- Ensure CGO is enabled: `go env CGO_ENABLED`

**GUI doesn't start:**
- Check display environment variables
- Verify Fyne dependencies are installed
- Run with debug flags: `./build.sh debug gui`

**Memory issues:**
- Monitor with: `./build.sh debug memory`
- Check available system memory
- Reduce image sizes for testing
