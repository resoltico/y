# Otsu Obliterator

A high-performance image processing application implementing advanced Otsu thresholding algorithms with a minimalist, text-only interface designed for 2025.

## Features

### Algorithms
- **2D Otsu Thresholding**: Advanced thresholding using both pixel intensity and neighborhood context for better noise handling
- **Iterative Triclass Thresholding**: Iterative approach that segments images into foreground, background, and "to-be-determined" regions

### Interface Design
- **Minimalist 2025 Design**: Clean, text-only interface following current design principles
- **Fixed 50/50 Split**: Original and Preview images displayed side-by-side in fixed ratio
- **Real-time Metrics**: PSNR and SSIM calculations displayed in status bar
- **Dynamic Parameters**: Algorithm-specific parameters exposed through radio buttons and sliders

### Technical Features
- **Hardware Acceleration**: OpenCV with general-purpose cross-platform acceleration
- **Memory Profiling**: Built-in Mat profiling and memory leak detection
- **Vectorized Operations**: Efficient OpenCV calls instead of pixel-by-pixel processing
- **Modular Architecture**: Clean separation of GUI, pipeline, algorithms, and debugging
- **Format Detection Debugging**: Comprehensive image format analysis and debugging

## Quick Start

### Prerequisites
- Go 1.24+
- OpenCV 4.11.0+
- Fyne v2.6+

### Installation
```bash
git clone <repository>
cd otsu-obliterator
make dev
```

### Running
```bash
# Development with profiling
make run-profile

# Debug format detection issues
make run-debug-format

# Production build
make prod && make run

# Memory leak detection
make check-leaks
```

## Usage

1. **Load Image**: Use OS-standard file dialogs (File > Open Image)
2. **Select Algorithm**: Choose between "2D Otsu" or "Iterative Triclass" via radio buttons
3. **Adjust Parameters**: Modify algorithm-specific parameters in the right panel
4. **Generate Preview**: Click "Generate Preview" to process the image
5. **View Metrics**: Check PSNR and SSIM values in the status bar
6. **Save Result**: Use OS-standard save dialog (File > Save Processed Image)

## Debugging Image Format Issues

The application includes comprehensive format detection debugging to help identify issues like the TIFF/PNG loading problem:

### Debug Features
- **Format Signature Analysis**: Examines file headers to identify actual format
- **URI Extension Detection**: Tracks extension from file URI
- **Standard Library Detection**: Logs Go's image package format detection
- **OpenCV Compatibility**: Verifies OpenCV IMDecode support
- **MimeType Validation**: Checks URI MimeType consistency

### Debug Logs to Monitor
When loading "Ut feritur ferit.tiff":
- `FormatDebug`: Format detection analysis with file signatures
- `ImageDebug`: Comprehensive image loading report
- Mismatch warnings if extension != detected format

### Running Debug Mode
```bash
# Focus on format detection
make run-debug-format

# View debug output in terminal for:
# - URI extension vs detected format
# - File signature analysis (hex dump)
# - Standard library vs OpenCV results
```

## Algorithm Parameters

### 2D Otsu Parameters
- **Quality**: Fast/Best processing mode
- **Window Size**: Neighborhood size (3-21, odd only)
- **Histogram Bins**: 2D histogram resolution (16-256)
- **Neighbourhood Metric**: mean/median/gaussian aggregation
- **Pixel Weight Factor**: Blend ratio between pixel intensity and neighborhood (0.0-1.0)
- **Smoothing Sigma**: Gaussian blur for histogram stabilization (0.0-5.0)
- **Use Log Histogram**: Log-scale histogram counts
- **Normalize Histogram**: Convert to probabilities
- **Apply Contrast Enhancement**: CLAHE preprocessing

### Iterative Triclass Parameters
- **Quality**: Fast/Best processing mode
- **Initial Threshold Method**: otsu/mean/median starting threshold
- **Histogram Bins**: Grayscale histogram resolution (16-256)
- **Convergence Epsilon**: Iteration stopping criterion (0.1-10.0)
- **Max Iterations**: Safety limit (1-20)
- **Minimum TBD Fraction**: Early stopping threshold (0.001-0.2)
- **Lower Upper Gap Factor**: TBD region adjustment (0.0-1.0)
- **Apply Preprocessing**: Contrast enhancement and denoising
- **Apply Cleanup**: Morphological filtering
- **Preserve Borders**: Edge preservation

## Architecture

### Modular Project Structure
```
otsu-obliterator/
├── main.go                   # Entry point (imports modular components)
├── app_core.go              # Application core and initialization
├── app_handlers.go          # Event handlers for user interactions
├── app_menus.go             # Menu setup and handlers
├── gui/                     # GUI components
│   ├── gui_main.go          # Main interface
│   ├── gui_parameters.go    # Parameter panel
│   └── gui_status.go        # Status bar
├── otsu/                    # Algorithm implementations
│   ├── otsu_manager.go      # Algorithm management
│   ├── otsu_2d.go           # 2D Otsu implementation
│   └── otsu_iterative.go    # Iterative Triclass implementation
├── pipeline/                # Image processing pipeline
│   ├── pipe_main.go         # Main pipeline structure
│   ├── pipe_loader.go       # Image loading with debug
│   ├── pipe_saver.go        # Image saving
│   ├── pipe_processor.go    # Algorithm processing
│   ├── pipe_metrics.go      # PSNR/SSIM calculations
│   └── pipe_conversion.go   # Image/Mat conversion
├── debug/                   # Debug and profiling
│   ├── debug_manager.go     # Debug management
│   ├── debug_image.go       # Image-specific debugging
│   ├── debug_format.go      # Format detection debugging
│   ├── debug_logging.go     # Debug logging (profiling builds)
│   └── debug_logging_release.go # Release logging
├── makefile                 # Build system with debug targets
├── go.mod                   # Go modules
└── FyneApp.toml            # Fyne configuration
```

### Design Principles
- **Modular**: Each component has a single responsibility with clear prefixes
- **Memory Safe**: Automatic Mat cleanup and profiling
- **Performance**: Vectorized operations and hardware acceleration
- **Maintainable**: Clear, descriptive code without subjective quality terms
- **Concurrent**: Uses fyne.Do for thread safety
- **Debuggable**: Comprehensive logging and format analysis

## Development

### Build Targets
```bash
make help                    # Show all available targets
make dev                     # Setup development environment
make build-profile           # Build with profiling
make run-profile             # Run with memory profiling
make run-debug-format        # Run with format debugging focus
make check-leaks             # Memory leak detection
make profile                 # Full profiling server
```

### Cross-Platform Builds
```bash
make build-windows           # Windows build
make build-macos             # macOS Intel build
make build-macos-arm64       # macOS Apple Silicon build
make build-linux             # Linux build
make build-macos-universal   # Universal macOS binary
```

### Debugging Format Issues
```bash
# Start with format debugging
make run-debug-format

# Load your TIFF file and monitor terminal for:
# [FormatDebug] Format Detection Analysis
# [ImageDebug] Image Load Debug Report
# [WARN] Format mismatch detected
```

The debug output will show:
- URI extension vs detected format
- File signature analysis (hex dump of first 16 bytes)
- Standard library detection result
- OpenCV IMDecode result
- Final format determination logic

### Profiling
Built-in profiling tracks:
- Mat creation/destruction
- Memory usage patterns
- Operation timing
- Format detection steps
- Performance bottlenecks

### Benchmarks
Run `make profile` and use Go's pprof tools:
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
go tool pprof http://localhost:6060/debug/pprof/gocv.io/x/gocv.Mat
```

## Troubleshooting

### TIFF Loading Issues
If "Ut feritur ferit.tiff" loads as PNG:

1. Run with `make run-debug-format`
2. Check terminal output for format mismatch warnings
3. Look for hex signature analysis in logs
4. Verify URI extension detection vs standard library result
5. Check if file is actually TIFF or misnamed PNG

### Common Debug Patterns
- Format signature mismatch: File extension doesn't match content
- URI extension empty: File dialog not preserving extension
- Standard library failure: Unsupported format variation
- OpenCV decode failure: Corrupted or unsupported image data

## Contributing

1. Follow the modular structure with appropriate prefixes
2. Use descriptive, non-subjective code comments
3. Maintain memory safety with Mat cleanup
4. Add debug logging for new features
5. Update documentation for changes

## License

[License information would go here]

## References

- [Otsu's Method - Wikipedia](https://en.wikipedia.org/wiki/Otsu%27s_method)
- [2D Otsu Research Papers](https://www.sciencedirect.com/science/article/abs/pii/S1047320316302206)
- [Iterative Triclass Thresholding](https://pubmed.ncbi.nlm.nih.gov/24474373/)
- [Fyne v2.6 Documentation](https://docs.fyne.io/api/v2.6/)
- [GoCV Documentation](https://pkg.go.dev/gocv.io/x/gocv)