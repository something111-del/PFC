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
                simple_vol = np.std(returns) * np.sqrt(252)
                # Ensure minimum volatility of 10% annualized
                return max(simple_vol, 0.10)
            
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
            
            # Apply minimum volatility floor of 2% and maximum of 100%
            volatility_annual = np.clip(volatility_annual, 0.02, 1.0)
            
            logger.info(f"GARCH volatility estimated: {volatility_annual:.4f}")
            return volatility_annual
            
        except Exception as e:
            logger.error(f"GARCH estimation failed: {str(e)}")
            # Fallback to simple volatility with minimum floor
            simple_vol = np.std(returns) * np.sqrt(252)
            return max(simple_vol, 0.15)  # 15% minimum for fallback
    
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
    
    def estimate_drift(self, returns: np.ndarray, use_exponential_weighting: bool = True) -> float:
        """
        Estimate drift (expected return) with exponential weighting
        
        Args:
            returns: Array of historical returns
            use_exponential_weighting: If True, recent returns get more weight
            
        Returns:
            Annualized drift
        """
        if len(returns) == 0:
            return 0.0
        
        if use_exponential_weighting and len(returns) > 5:
            # Exponentially weighted mean (recent data matters more)
            # Decay factor: 0.94 means yesterday has 94% weight of today
            weights = np.exp(np.linspace(-1, 0, len(returns)))
            weights = weights / weights.sum()
            mean_return = np.average(returns, weights=weights)
        else:
            # Simple mean return
            mean_return = np.mean(returns)
        
        # Annualize (252 trading days)
        drift_annual = mean_return * 252
        
        # Clip extreme drifts to realistic range (-50% to +100% annually)
        drift_annual = np.clip(drift_annual, -0.50, 1.0)
        
        logger.info(f"Estimated drift: {drift_annual:.4f} ({drift_annual*100:.2f}% annually)")
        
        return drift_annual
