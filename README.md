## Architecture

### Core Models
- **Record**: Represents a CSV row with metadata (line number, filename)
- **Result**: Processing outcome with status, error, and timing
- **Summary**: Aggregated statistics across all records

### Interfaces
- **Processor**: Contract for record transformation logic
- **ResultHandler**: Contract for handling processed results