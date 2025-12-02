# PFC - Portfolio Forecast Chrome Extension

PFC is a powerful Chrome extension that provides real-time, 24-hour portfolio value forecasting directly on your favorite brokerage websites.

## 🚀 Features

- **Smart Detection**: Automatically detects portfolio holdings on Robinhood, Webull, TD Ameritrade, E*TRADE, Fidelity, and Charles Schwab.
- **Advanced Forecasting**: Uses GARCH(1,1) for volatility estimation and Monte Carlo simulations (10,000 paths) for price prediction.
- **Risk Analysis**: Color-coded risk indicators (Green/Yellow/Red) based on expected return and volatility.
- **Privacy First**: No portfolio data is stored permanently. Calculations are done on-demand.

## 🏗 Architecture

- **Chrome Extension**: Frontend interface (HTML/CSS/JS)
- **Go API Service**: High-performance orchestrator (Fiber, concurrency)
- **Python Model Service**: Forecasting engine (FastAPI, NumPy, Arch)
- **Google Cloud**: Cloud Run, Firestore, Cloud Storage

## 🛠 Setup & Deployment

### Prerequisites
- Google Cloud Project
- GitHub Repository
- Alpha Vantage API Key

### Local Development
1. Clone the repository
2. Run services using Docker Compose (optional) or individually
3. Load extension in Chrome Developer Mode

### Cloud Deployment
The project uses GitHub Actions for CI/CD. Pushing to `main` triggers automatic deployment to Cloud Run.

See [DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed instructions.

## 📦 Chrome Web Store

To package the extension for the Web Store:
```bash
./infrastructure/scripts/package-extension.sh
```
Upload the generated ZIP file to the Chrome Developer Dashboard.

## 📄 License

MIT License
