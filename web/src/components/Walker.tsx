import React, { useEffect, useRef } from 'react';

interface WalkerProps {
  width?: number;
  height?: number;
}

const Walker: React.FC<WalkerProps> = ({
  width = window.innerWidth,
  height = window.innerHeight,
}) => {
  const walkerCanvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    if (!walkerCanvasRef.current) return;
    const walkerCanvas = walkerCanvasRef.current;
    const walkerContext = walkerCanvas.getContext('2d');
    if (!walkerContext) return;

    walkerCanvas.width = width * 2;
    walkerCanvas.height = height * 2;
    walkerCanvas.style.width = `${width}px`;
    walkerCanvas.style.height = `${height}px`;
    walkerContext.scale(2, 2);

    const halfWidth = width;
    const halfHeight = height;

    walker();

    function walker() {
      const x = halfWidth / 2;
      const y = halfHeight / 2;
      const stepSize = 10;
      const walkerCount = 5;
      const angles = [0, Math.PI / 2, Math.PI, (3 * Math.PI) / 2];
      const colors = ['#B08D57', '#D4AF37', '#C9B037', '#5F9EA0', '#4682B4'];

      if (!walkerContext) return;

      for (let i = 0; i < walkerCount; i++) {
        walkingCircle(x, y, stepSize, i);
      }

      function walkingCircle(x: number, y: number, stepSize: number, color: number) {
        let prevX = x;
        let prevY = y;

        draw();

        function draw() {
          const angle = pick(angles);
          prevX = x;
          prevY = y;
          x += Math.cos(angle ?? 0) * stepSize;
          y += Math.sin(angle ?? 0) * stepSize;

          if (x < 0) x = 0;
          if (x > halfWidth) x = halfWidth;
          if (y < 0) y = 0;
          if (y > halfHeight) y = halfHeight;

          if (!walkerContext) return;

          walkerContext.beginPath();
          walkerContext.moveTo(prevX, prevY);
          walkerContext.lineTo(x, y);
          walkerContext.strokeStyle = colors[color % colors.length];
          walkerContext.lineWidth = 2;
          walkerContext.stroke();

          requestAnimationFrame(draw);
        }

        function rangeFloor(min: number, max: number): number {
          return Math.floor(Math.random() * (max - min) + min);
        }

        function pick<T>(array: T[]): T | undefined {
          if (array.length === 0) return undefined;
          return array[rangeFloor(0, array.length)];
        }
      }
    }
  }, [width, height]);

  return <canvas ref={walkerCanvasRef}></canvas>;
};

export default Walker;
