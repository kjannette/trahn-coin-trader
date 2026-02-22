import { Chart } from '../components/Chart';
import { CarouselNav, DayIndicators, ConnectionStatus } from '../components/CarouselNav';
import { Stats } from '../components/Stats';
import { useTradingData } from '../contexts/TradingDataContext';
import { useCarousel } from '../hooks/useTradingData';
import { useHistoricalDay } from '../hooks/useTradingData';
import { useAuth } from '../contexts/AuthContext';
import { useEffect } from 'react';

export function Dashboard() {
    const { logout } = useAuth();
    const { state, dispatch } = useTradingData();
    const carousel = useCarousel();
    
    // Load historical day data when viewing past days
    const historicalDate = !state.isLive ? state.availableDays[state.currentDayIndex] : null;
    const { data: historicalData } = useHistoricalDay(historicalDate, !state.isLive);
    
    // Update context with historical data when loaded
    useEffect(() => {
        if (historicalData && !state.isLive) {
            dispatch({
                type: 'LOAD_HISTORICAL_DAY',
                prices: historicalData.prices,
                trades: historicalData.trades,
                index: state.currentDayIndex,
            });
        }
    }, [historicalData, state.isLive, state.currentDayIndex, dispatch]);
    
    const {
        prices,
        trades,
        grid,
        availableDays,
        currentDayIndex,
        isLive,
        connectionStatus,
    } = state;
    
    const {
        navigatePrev,
        navigateNext,
        navigateTo,
        canGoPrev,
        canGoNext,
    } = carousel;
    
    const currentDay = availableDays[currentDayIndex] || null;
    const loading = connectionStatus === 'connecting';

    return (
        <div className="container">
            <header>
                <pre className="ascii-logo">{`
   ████████╗██████╗  █████╗ ██╗  ██╗███╗   ██╗
   ╚══██╔══╝██╔══██╗██╔══██╗██║  ██║████╗  ██║
      ██║   ██████╔╝███████║███████║██╔██╗ ██║
      ██║   ██╔══██╗██╔══██║██╔══██║██║╚██╗██║
      ██║   ██║  ██║██║  ██║██║  ██║██║ ╚████║
      ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝`}</pre>
                <button className="logout-btn" onClick={logout}>Logout</button>
            </header>
            
            <CarouselNav 
                currentDay={currentDay}
                availableDays={availableDays}
                currentDayIndex={currentDayIndex}
                isLive={isLive}
                canGoPrev={canGoPrev}
                canGoNext={canGoNext}
                onPrev={navigatePrev}
                onNext={navigateNext}
                onNavigateTo={navigateTo}
            />
            
            <div className="chart-wrapper">
                <Chart prices={prices} trades={trades} />
                {loading && (
                    <div className="loading-overlay">
                        <div className="spinner" />
                        <span>Waiting for data...</span>
                    </div>
                )}
            </div>
            
            <Stats prices={prices} trades={trades} />
            
            <DayIndicators 
                availableDays={availableDays}
                currentDayIndex={currentDayIndex}
                onNavigateTo={navigateTo}
            />
            
            <div className="legend">
                <span className="legend-item">
                    <span className="dot buy-dot" /> Buy (USDC → ETH)
                </span>
                <span className="legend-item">
                    <span className="dot sell-dot" /> Sell (ETH → USDC)
                </span>
            </div>
            
            {grid && grid.length > 0 && (
                <div className="grid-levels">
                    <div className="grid-levels-header">Grid Levels</div>
                    <div className="grid-levels-container">
                        <div className="grid-column">
                            <div className="grid-column-header buy">Buy Levels</div>
                            {grid
                                .filter(level => level.side === 'buy')
                                .sort((a, b) => b.price - a.price)
                                .map((level, idx) => (
                                    <div key={idx} className={`grid-level buy ${level.filled ? 'filled' : ''}`}>
                                        ${level.price.toFixed(2)}
                                    </div>
                                ))}
                        </div>
                        <div className="grid-column">
                            <div className="grid-column-header sell">Sell Levels</div>
                            {grid
                                .filter(level => level.side === 'sell')
                                .sort((a, b) => a.price - b.price)
                                .map((level, idx) => (
                                    <div key={idx} className={`grid-level sell ${level.filled ? 'filled' : ''}`}>
                                        ${level.price.toFixed(2)}
                                    </div>
                                ))}
                        </div>
                    </div>
                </div>
            )}
            
            <ConnectionStatus status={connectionStatus} />
        </div>
    );
}

