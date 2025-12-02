import numpy as np
import logging

logger = logging.getLogger(__name__)

class MonteCarloSimulator:
    """
    Monte Carlo simulation for price path generation
    """
    
    def __init__(self, num_simulations: int = 10000, forecast_hours: int = 24):
        self.num_simulations = num_simulations
        self.forecast_hours = forecast_hours
        # Convert hours to trading day fraction (assuming 6.5 trading hours per day)
        self.time_fraction = forecast_hours / (6.5 * 252)
    
    def simulate_paths(
        self,
        current_price: float,
        drift: float,
        volatility: float
    ) -> np.ndarray:
        """
        Generate Monte Carlo simulation paths using Geometric Brownian Motion
        
        Args:
            current_price: Current stock price
            drift: Expected return (annualized)
            volatility: Volatility (annualized)
            
        Returns:
            Array of simulated final prices (num_simulations,)
        """
        try:
            # Adjust drift and volatility for time horizon
            dt = self.time_fraction
            drift_adjusted = drift * dt
            vol_adjusted = volatility * np.sqrt(dt)
            
            # Generate random normal samples
            random_shocks = np.random.normal(0, 1, self.num_simulations)
            
            # Geometric Brownian Motion formula
            # S_t = S_0 * exp((drift - 0.5 * vol^2) * dt + vol * sqrt(dt) * Z)
            price_changes = np.exp(
                (drift_adjusted - 0.5 * vol_adjusted ** 2) +
                vol_adjusted * random_shocks
            )
            
            final_prices = current_price * price_changes
            
            logger.info(f"Generated {self.num_simulations} simulation paths")
            return final_prices
            
        except Exception as e:
            logger.error(f"Monte Carlo simulation failed: {str(e)}")
            # Return array with current price (no change scenario)
            return np.full(self.num_simulations, current_price)
    
    def calculate_percentiles(self, simulated_prices: np.ndarray) -> dict:
        """
        Calculate percentiles from simulated prices
        
        Args:
            simulated_prices: Array of simulated final prices
            
        Returns:
            Dictionary with p5, p50, p95 percentiles
        """
        percentiles = {
            'p5': float(np.percentile(simulated_prices, 5)),
            'p50': float(np.percentile(simulated_prices, 50)),
            'p95': float(np.percentile(simulated_prices, 95))
        }
        
        logger.info(f"Percentiles - 5th: ${percentiles['p5']:.2f}, "
                   f"50th: ${percentiles['p50']:.2f}, "
                   f"95th: ${percentiles['p95']:.2f}")
        
        return percentiles
    
    def calculate_risk_metrics(
        self,
        current_price: float,
        percentiles: dict,
        volatility: float
    ) -> dict:
        """
        Calculate risk metrics and determine risk level
        
        Args:
            current_price: Current stock price
            percentiles: Dictionary with p5, p50, p95
            volatility: Estimated volatility
            
        Returns:
            Dictionary with risk metrics and risk level
        """
        expected_return = (percentiles['p50'] - current_price) / current_price
        downside_risk = (current_price - percentiles['p5']) / current_price
        upside_potential = (percentiles['p95'] - current_price) / current_price
        
        # Determine risk color
        # Green: Positive expected return and low volatility
        # Red: Negative expected return or high volatility
        # Yellow: Moderate risk
        
        if expected_return > 0.02 and volatility < 0.05:
            risk_level = "green"
        elif expected_return < -0.02 or volatility > 0.10:
            risk_level = "red"
        else:
            risk_level = "yellow"
        
        return {
            'risk': risk_level,
            'expected_return': expected_return,
            'downside_risk': downside_risk,
            'upside_potential': upside_potential,
            'volatility': volatility
        }
