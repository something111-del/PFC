import numpy as np
import logging
from typing import List, Dict

from app.models.garch import GARCHModel
from app.models.monte_carlo import MonteCarloSimulator
from app.schemas.request import TickerForecast, Percentiles

logger = logging.getLogger(__name__)

class ForecastService:
    """
    Main forecasting service combining GARCH and Monte Carlo
    """
    
    def __init__(self, num_simulations: int = 10000, forecast_hours: int = 24):
        self.garch_model = GARCHModel()
        self.monte_carlo = MonteCarloSimulator(num_simulations, forecast_hours)
        logger.info(f"Forecast service initialized: {num_simulations} simulations, {forecast_hours}h horizon")
    
    def generate_forecast(
        self,
        tickers: List[str],
        historical_data: Dict[str, List[float]]
    ) -> dict:
        """
        Generate forecasts for multiple tickers
        
        Args:
            tickers: List of ticker symbols
            historical_data: Dictionary mapping ticker to historical prices
            
        Returns:
            Dictionary with forecasts and overall risk
        """
        forecasts = []
        risk_scores = []
        
        for ticker in tickers:
            try:
                forecast = self._forecast_single_ticker(ticker, historical_data.get(ticker, []))
                forecasts.append(forecast)
                risk_scores.append(self._risk_to_score(forecast.risk))
            except Exception as e:
                logger.error(f"Failed to forecast {ticker}: {str(e)}")
                # Add a default forecast
                forecasts.append(self._default_forecast(ticker))
                risk_scores.append(2)  # Yellow risk
        
        # Calculate overall portfolio risk
        avg_risk_score = np.mean(risk_scores) if risk_scores else 2
        overall_risk = self._score_to_risk(avg_risk_score)
        
        return {
            'forecasts': [f.dict() for f in forecasts],
            'risk': overall_risk
        }
    
    def _forecast_single_ticker(
        self,
        ticker: str,
        historical_prices: List[float]
    ) -> TickerForecast:
        """
        Generate forecast for a single ticker
        
        Args:
            ticker: Ticker symbol
            historical_prices: List of historical prices
            
        Returns:
            TickerForecast object
        """
        logger.info(f"Forecasting {ticker} with {len(historical_prices)} data points")
        
        if len(historical_prices) < 2:
            logger.warning(f"Insufficient data for {ticker}, using default forecast")
            return self._default_forecast(ticker)
        
        # Convert to numpy array
        prices = np.array(historical_prices)
        current_price = float(prices[-1])
        
        # Step 1: Calculate returns
        returns = self.garch_model.calculate_returns(prices)
        
        if len(returns) == 0:
            return self._default_forecast(ticker, current_price)
        
        # Step 2: Estimate volatility using GARCH
        volatility = self.garch_model.estimate_volatility(returns)
        
        # Step 3: Estimate drift
        drift = self.garch_model.estimate_drift(returns)
        
        # Step 4: Run Monte Carlo simulation
        simulated_prices = self.monte_carlo.simulate_paths(
            current_price=current_price,
            drift=drift,
            volatility=volatility
        )
        
        # Step 5: Calculate percentiles
        percentiles_dict = self.monte_carlo.calculate_percentiles(simulated_prices)
        
        # Step 6: Calculate risk metrics
        risk_metrics = self.monte_carlo.calculate_risk_metrics(
            current_price=current_price,
            percentiles=percentiles_dict,
            volatility=volatility
        )
        
        # Build response
        forecast = TickerForecast(
            symbol=ticker,
            currentPrice=current_price,
            forecast=Percentiles(**percentiles_dict),
            volatility=volatility,
            risk=risk_metrics['risk']
        )
        
        logger.info(f"{ticker} forecast complete: Risk={forecast.risk}, Vol={volatility:.4f}")
        return forecast
    
    def _default_forecast(self, ticker: str, current_price: float = 100.0) -> TickerForecast:
        """
        Generate a default forecast when data is insufficient
        """
        return TickerForecast(
            symbol=ticker,
            currentPrice=current_price,
            forecast=Percentiles(
                p5=current_price * 0.98,
                p50=current_price,
                p95=current_price * 1.02
            ),
            volatility=0.15,  # Default 15% volatility
            risk="yellow"
        )
    
    def _risk_to_score(self, risk: str) -> int:
        """Convert risk level to numeric score"""
        mapping = {'green': 1, 'yellow': 2, 'red': 3}
        return mapping.get(risk, 2)
    
    def _score_to_risk(self, score: float) -> str:
        """Convert numeric score to risk level"""
        if score < 1.5:
            return 'green'
        elif score < 2.5:
            return 'yellow'
        else:
            return 'red'
