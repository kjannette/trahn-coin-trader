export function Stats({ prices, trades }) {
    if (!prices || prices.length === 0) {
        return (
            <div className="stats">
                <Stat label="Current Price" value="--" />
                <Stat label="24H High" value="--" className="high" />
                <Stat label="24H Low" value="--" className="low" />
                <Stat label="Buys" value="0" className="buy" />
                <Stat label="Sells" value="0" className="sell" />
            </div>
        );
    }

    const priceValues = prices.map(d => d.p || d.price);
    const currentPrice = priceValues[priceValues.length - 1];
    const highPrice = Math.max(...priceValues);
    const lowPrice = Math.min(...priceValues);
    
    const buyCount = (trades || []).filter(t => t.side === 'buy').length;
    const sellCount = (trades || []).filter(t => t.side === 'sell').length;

    const formatPrice = (price) => {
        return '$' + price.toLocaleString('en-US', { 
            minimumFractionDigits: 2, 
            maximumFractionDigits: 2 
        });
    };

    return (
        <div className="stats">
            <Stat label="Current Price" value={formatPrice(currentPrice)} />
            <Stat label="24H High" value={formatPrice(highPrice)} className="high" />
            <Stat label="24H Low" value={formatPrice(lowPrice)} className="low" />
            <Stat label="Buys" value={buyCount} className="buy" />
            <Stat label="Sells" value={sellCount} className="sell" />
        </div>
    );
}

function Stat({ label, value, className = '' }) {
    return (
        <div className="stat">
            <span className="stat-label">{label}</span>
            <span className={`stat-value ${className}`}>{value}</span>
        </div>
    );
}

