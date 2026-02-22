import { useRef, useEffect, useState } from 'react';

const COLORS = {
    line: '#58a6ff',
    green: '#3fb950',
    greenFill: 'rgba(63, 185, 80, 0.12)',
    red: '#f85149',
    redFill: 'rgba(248, 81, 73, 0.12)',
    grid: '#21262d',
    text: '#8b949e',
    textMuted: '#484f58',
    baseline: '#30363d',
    buy: '#f0c000',
    sell: '#db6d28',
    bg: '#0d1117',
};

const PADDING = { top: 20, right: 80, bottom: 40, left: 20 };

export function Chart({ prices, trades }) {
    const canvasRef = useRef(null);
    const containerRef = useRef(null);
    const [tooltip, setTooltip] = useState(null);
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });

    // Handle resize
    useEffect(() => {
        const updateDimensions = () => {
            if (containerRef.current) {
                const rect = containerRef.current.getBoundingClientRect();
                setDimensions({ width: rect.width, height: rect.height });
            }
        };
        
        updateDimensions();
        window.addEventListener('resize', updateDimensions);
        return () => window.removeEventListener('resize', updateDimensions);
    }, []);

    // Draw chart
    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas || dimensions.width === 0) return;
        
        const ctx = canvas.getContext('2d');
        const dpr = window.devicePixelRatio || 1;
        
        canvas.width = dimensions.width * dpr;
        canvas.height = dimensions.height * dpr;
        ctx.scale(dpr, dpr);
        
        const width = dimensions.width;
        const height = dimensions.height;
        const chartWidth = width - PADDING.left - PADDING.right;
        const chartHeight = height - PADDING.top - PADDING.bottom;
        
        // Clear
        ctx.clearRect(0, 0, width, height);
        
        if (!prices || prices.length === 0) {
            ctx.fillStyle = COLORS.textMuted;
            ctx.font = '14px JetBrains Mono, monospace';
            ctx.textAlign = 'center';
            ctx.fillText('No data for this day', width / 2, height / 2);
            return;
        }
        
        // Calculate bounds
        const priceValues = prices.map(d => d.p || d.price);
        const times = prices.map(d => d.t || d.timestamp);
        
        let minPrice = Math.min(...priceValues);
        let maxPrice = Math.max(...priceValues);
        const minTime = Math.min(...times);
        const maxTime = Math.max(...times);
        
        const pricePadding = (maxPrice - minPrice) * 0.05 || 10;
        minPrice -= pricePadding;
        maxPrice += pricePadding;
        
        const baselinePrice = priceValues[0];
        
        // Coordinate converters
        const priceToY = (price) => {
            const range = maxPrice - minPrice;
            const normalized = (price - minPrice) / range;
            return PADDING.top + chartHeight * (1 - normalized);
        };
        
        const timeToX = (timestamp) => {
            const range = maxTime - minTime;
            const normalized = (timestamp - minTime) / range;
            return PADDING.left + chartWidth * normalized;
        };
        
        // Draw grid
        ctx.strokeStyle = COLORS.grid;
        ctx.lineWidth = 1;
        
        const priceStep = calculateNiceStep(maxPrice - minPrice, 5);
        const startPrice = Math.ceil(minPrice / priceStep) * priceStep;
        
        for (let price = startPrice; price <= maxPrice; price += priceStep) {
            const y = priceToY(price);
            ctx.beginPath();
            ctx.moveTo(PADDING.left, y);
            ctx.lineTo(width - PADDING.right, y);
            ctx.stroke();
        }
        
        // Draw baseline
        const baselineY = priceToY(baselinePrice);
        ctx.strokeStyle = COLORS.baseline;
        ctx.setLineDash([4, 4]);
        ctx.beginPath();
        ctx.moveTo(PADDING.left, baselineY);
        ctx.lineTo(width - PADDING.right, baselineY);
        ctx.stroke();
        ctx.setLineDash([]);
        
        // Draw gradient fill
        if (prices.length >= 2) {
            ctx.beginPath();
            ctx.moveTo(timeToX(times[0]), baselineY);
            ctx.lineTo(timeToX(times[0]), priceToY(priceValues[0]));
            
            for (let i = 1; i < prices.length; i++) {
                ctx.lineTo(timeToX(times[i]), priceToY(priceValues[i]));
            }
            
            ctx.lineTo(timeToX(times[times.length - 1]), baselineY);
            ctx.closePath();
            
            const gradient = ctx.createLinearGradient(0, PADDING.top, 0, height - PADDING.bottom);
            gradient.addColorStop(0, COLORS.greenFill);
            gradient.addColorStop(0.5, 'rgba(0,0,0,0)');
            gradient.addColorStop(1, COLORS.redFill);
            ctx.fillStyle = gradient;
            ctx.fill();
        }
        
        // Draw price line
        if (prices.length >= 2) {
            ctx.beginPath();
            ctx.strokeStyle = COLORS.line;
            ctx.lineWidth = 2;
            ctx.lineJoin = 'round';
            ctx.lineCap = 'round';
            
            ctx.moveTo(timeToX(times[0]), priceToY(priceValues[0]));
            for (let i = 1; i < prices.length; i++) {
                ctx.lineTo(timeToX(times[i]), priceToY(priceValues[i]));
            }
            ctx.stroke();
        }
        
        // Draw Y axis labels
        ctx.fillStyle = COLORS.text;
        ctx.font = '11px JetBrains Mono, monospace';
        ctx.textAlign = 'left';
        
        for (let price = startPrice; price <= maxPrice; price += priceStep) {
            const y = priceToY(price);
            ctx.fillText(formatPrice(price), width - PADDING.right + 8, y + 4);
        }
        
        // Draw X axis labels
        ctx.textAlign = 'center';
        const hourMs = 60 * 60 * 1000;
        const rangeMs = maxTime - minTime;
        let stepMs = hourMs * 4;
        if (rangeMs < hourMs * 6) stepMs = hourMs;
        if (rangeMs < hourMs * 2) stepMs = hourMs / 2;
        
        let current = Math.ceil(minTime / stepMs) * stepMs;
        while (current <= maxTime) {
            const x = timeToX(current);
            if (x >= PADDING.left && x <= width - PADDING.right) {
                const date = new Date(current);
                const label = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
                ctx.fillText(label, x, height - PADDING.bottom + 20);
            }
            current += stepMs;
        }
        
        // Draw trades
        for (const trade of (trades || [])) {
            const x = timeToX(trade.t || trade.timestamp);
            const y = priceToY(trade.price);
            
            if (x < PADDING.left || x > width - PADDING.right) continue;
            
            const color = trade.side === 'buy' ? COLORS.buy : COLORS.sell;
            const radius = 7;
            
            // Glow
            ctx.beginPath();
            ctx.arc(x, y, radius + 4, 0, Math.PI * 2);
            ctx.fillStyle = color + '33';
            ctx.fill();
            
            // Outer
            ctx.beginPath();
            ctx.arc(x, y, radius, 0, Math.PI * 2);
            ctx.fillStyle = color;
            ctx.fill();
            
            // Inner
            ctx.beginPath();
            ctx.arc(x, y, radius - 3, 0, Math.PI * 2);
            ctx.fillStyle = COLORS.bg;
            ctx.fill();
        }
        
        // Draw current price tag
        if (prices.length > 0) {
            const lastPrice = priceValues[priceValues.length - 1];
            const y = priceToY(lastPrice);
            const x = width - PADDING.right;
            
            const isUp = lastPrice >= baselinePrice;
            const color = isUp ? COLORS.green : COLORS.red;
            
            ctx.fillStyle = color;
            roundRect(ctx, x + 4, y - 11, 70, 22, 4);
            ctx.fill();
            
            ctx.fillStyle = '#fff';
            ctx.font = 'bold 11px JetBrains Mono, monospace';
            ctx.textAlign = 'left';
            ctx.fillText(formatPrice(lastPrice), x + 10, y + 4);
        }
        
    }, [prices, trades, dimensions]);

    // Mouse handling for tooltip
    const handleMouseMove = (e) => {
        if (!prices || prices.length === 0) {
            setTooltip(null);
            return;
        }
        
        const rect = canvasRef.current.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;
        
        const width = dimensions.width;
        const height = dimensions.height;
        const chartWidth = width - PADDING.left - PADDING.right;
        
        if (x < PADDING.left || x > width - PADDING.right ||
            y < PADDING.top || y > height - PADDING.bottom) {
            setTooltip(null);
            return;
        }
        
        const times = prices.map(d => d.t || d.timestamp);
        const minTime = Math.min(...times);
        const maxTime = Math.max(...times);
        
        const normalized = (x - PADDING.left) / chartWidth;
        const timestamp = minTime + normalized * (maxTime - minTime);
        
        let nearestPoint = null;
        let nearestDistance = Infinity;
        
        for (const point of prices) {
            const t = point.t || point.timestamp;
            const distance = Math.abs(t - timestamp);
            if (distance < nearestDistance) {
                nearestDistance = distance;
                nearestPoint = point;
            }
        }
        
        if (nearestPoint) {
            setTooltip({
                x: e.clientX - rect.left + 15,
                y: e.clientY - rect.top - 10,
                price: nearestPoint.p || nearestPoint.price,
                timestamp: nearestPoint.t || nearestPoint.timestamp,
            });
        }
    };

    return (
        <div className="chart-container" ref={containerRef}>
            <canvas 
                ref={canvasRef}
                style={{ width: '100%', height: '100%' }}
                onMouseMove={handleMouseMove}
                onMouseLeave={() => setTooltip(null)}
            />
            {tooltip && (
                <div 
                    className="tooltip visible"
                    style={{ left: tooltip.x, top: tooltip.y }}
                >
                    <div className="price">
                        ${tooltip.price.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}
                    </div>
                    <div className="time">
                        {new Date(tooltip.timestamp).toLocaleString('en-US', {
                            month: 'short',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit'
                        })}
                    </div>
                </div>
            )}
        </div>
    );
}

function calculateNiceStep(range, targetSteps) {
    const roughStep = range / targetSteps;
    const magnitude = Math.pow(10, Math.floor(Math.log10(roughStep)));
    const normalized = roughStep / magnitude;
    
    let niceStep;
    if (normalized <= 1.5) niceStep = 1;
    else if (normalized <= 3) niceStep = 2;
    else if (normalized <= 7) niceStep = 5;
    else niceStep = 10;
    
    return niceStep * magnitude;
}

function formatPrice(price) {
    if (price >= 10000) {
        return '$' + (price / 1000).toFixed(1) + 'k';
    }
    return '$' + price.toLocaleString('en-US', { minimumFractionDigits: 0, maximumFractionDigits: 0 });
}

function roundRect(ctx, x, y, width, height, radius) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
    ctx.closePath();
}

