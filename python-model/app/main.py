from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import uvicorn
import os
from typing import List, Dict
import logging

from app.services.forecast_service import ForecastService
from app.schemas.request import ForecastRequest, HealthResponse

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="PFC Forecasting Service",
    description="GARCH + Monte Carlo portfolio forecasting",
    version="1.1.0"
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Initialize forecast service
forecast_service = ForecastService(
    num_simulations=int(os.getenv("NUM_SIMULATIONS", "10000")),
    forecast_hours=int(os.getenv("FORECAST_HOURS", "24"))
)

@app.get("/", response_model=Dict[str, str])
async def root():
    return {
        "service": "PFC Forecasting Service",
        "version": "1.1.0",
        "status": "running"
    }

@app.get("/health", response_model=HealthResponse)
async def health():
    return HealthResponse(
        status="healthy",
        service="pfc-python-model",
        version="1.1.0"
    )

@app.post("/predict")
async def predict(request: ForecastRequest):
    """
    Generate 24-hour forecast using GARCH + Monte Carlo
    """
    try:
        logger.info(f"Received forecast request for {len(request.tickers)} tickers")
        
        # Generate forecasts
        result = forecast_service.generate_forecast(
            tickers=request.tickers,
            historical_data=request.historicalData
        )
        
        logger.info(f"Forecast generated successfully. Risk: {result['risk']}")
        return result
        
    except Exception as e:
        logger.error(f"Forecast generation failed: {str(e)}")
        raise HTTPException(status_code=500, detail=str(e))

if __name__ == "__main__":
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)
