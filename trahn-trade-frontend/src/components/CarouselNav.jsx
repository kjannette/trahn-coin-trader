export function CarouselNav({ 
    currentDay, 
    availableDays, 
    currentDayIndex,
    isLive, 
    canGoPrev, 
    canGoNext, 
    onPrev, 
    onNext,
    onNavigateTo 
}) {
    const formatDate = (day) => {
        if (!day) return 'No Data';
        const date = new Date(day + 'T12:00:00');
        return date.toLocaleDateString('en-US', {
            weekday: 'short',
            month: 'short',
            day: 'numeric',
            year: 'numeric'
        });
    };

    return (
        <div className="carousel-nav">
            <button 
                className="nav-arrow" 
                onClick={onPrev} 
                disabled={!canGoPrev}
                title="Previous Day"
            >
                ◀
            </button>
            
            <div className="date-display">
                <span className="date-label">{formatDate(currentDay)}</span>
                <span className={`date-status ${isLive ? 'live' : ''}`}>
                    {isLive ? 'LIVE' : '24h snapshot'}
                </span>
            </div>
            
            <button 
                className="nav-arrow" 
                onClick={onNext} 
                disabled={!canGoNext}
                title="Next Day"
            >
                ▶
            </button>
        </div>
    );
}

export function DayIndicators({ availableDays, currentDayIndex, onNavigateTo }) {
    return (
        <div className="day-indicators">
            {availableDays.map((day, i) => (
                <div 
                    key={day}
                    className={`day-dot ${i === currentDayIndex ? 'active' : ''}`}
                    title={day}
                    onClick={() => onNavigateTo(i)}
                />
            ))}
        </div>
    );
}

export function ConnectionStatus({ status }) {
    const statusText = {
        connected: 'Live',
        waiting: 'Waiting for backend...',
        connecting: 'Connecting...',
    };

    return (
        <div className={`connection-status ${status}`}>
            <span className="status-dot" />
            <span className="status-text">{statusText[status] || 'Unknown'}</span>
        </div>
    );
}

