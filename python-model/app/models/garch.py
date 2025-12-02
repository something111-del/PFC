import numpy as np
from arch import arch_model
import logging

logger = logging.getLogger(__name__)

class GARCHModel:
    """
    GARCH(1,1) model for volatility estimation
    """
    
    def __init__(self):
        self.model = None
        self.fitted_model = None
    
    def estimate_volatility(self, returns: np.ndarray) -> float:
        """
        Estimate volatility using GARCH(1,1)
        
        Args:
            returns: Array of historical returns
            
        Returns:
            Estimated volatility (annualized)
        """
        try:
            if len(returns) < 30:
                # Fallback to simple standard deviation if not enough data
                logger.warning("Insufficient data for GARCH, using simple volatility")
                return np.std(returns) * np.sqrt(252)  # Annualized
            
            # Scale returns to percentage
            returns_pct = returns * 100
            
            # Fit GARCH(1,1) model
            self.model = arch_model(
                returns_pct,
                vol='Garch',
                p=1,
                q=1,
                rescale=False
            )
            
            self.fitted_model = self.model.fit(disp='off', show_warning=False)
            
            # Get conditional volatility forecast
            forecast = self.fitted_model.forecast(horizon=1)
            volatility = np.sqrt(forecast.variance.values[-1, 0])
            
            # Convert back to decimal and annualize
            volatility_annual = (volatility / 100) * np.sqrt(252)
            
            logger.info(f"GARCH volatility estimated: {volatility_annual:.4f}")
            return volatility_annual
            
        except Exception as e:
            logger.error(f"GARCH estimation failed: {str(e)}")
            # Fallback to simple volatility
            return np.std(returns) * np.sqrt(252)
    
    def calculate_returns(self, prices: np.ndarray) -> np.ndarray:
        """
        Calculate log returns from prices
        
        Args:
            prices: Array of historical prices
            
        Returns:
            Array of log returns
        """
        if len(prices) < 2:
            return np.array([])
        
        returns = np.diff(np.log(prices))
        return returns
    
    def estimate_drift(self, returns: np.ndarray) -> float:
        """
        Estimate drift (expected return)
        
        Args:
            returns: Array of historical returns
            
        Returns:
            Annualized drift
        """
        if len(returns) == 0:
            return 0.0
        
        # Mean return
        mean_return = np.mean(returns)
        
        # Annualize (252 trading days)
        drift_annual = mean_return * 252
        
        return drift_annual
