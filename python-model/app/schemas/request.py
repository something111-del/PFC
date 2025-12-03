from pydantic import BaseModel, Field
from typing import List, Dict, Optional

class ForecastRequest(BaseModel):
    tickers: List[str] = Field(..., min_items=1, max_items=50)
    currentPrices: Dict[str, float] = Field(default_factory=dict)
    historicalData: Dict[str, List[float]] = Field(default_factory=dict)

class Percentiles(BaseModel):
    p5: float
    p50: float
    p95: float

class TickerForecast(BaseModel):
    symbol: str
    currentPrice: float
    forecast: Percentiles
    volatility: float
    risk: str  # "green", "yellow", "red"

class ForecastResponse(BaseModel):
    forecasts: List[TickerForecast]
    risk: str

class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
