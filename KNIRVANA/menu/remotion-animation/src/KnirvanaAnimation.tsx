import { AbsoluteFill, interpolate, useCurrentFrame, Sequence } from 'remotion';
import { Star, Zap } from 'lucide-react';

export const KnirvanaAnimation = () => {
  const frame = useCurrentFrame();
  const duration = 300;

  const starsOpacity = interpolate(frame, [0, 60], [0, 1]);
  const supernovaScale = interpolate(frame, [60, 90], [0.1, 3]);
  const siriusScale = interpolate(frame, [90, 120, 150], [0, 1.2, 0.8]);
  const siriusRotation = interpolate(frame, [90, 150], [0, 360]);
  const siriusCoreOpacity = interpolate(frame, [120, 150], [1, 0.3]);
  const electricSparksOpacity = interpolate(frame, [120, 150], [0, 1]);
  const menuScale = interpolate(frame, [150, 220], [0, 1]);
  const menuOpacity = interpolate(frame, [150, 180], [0, 1]);
  const iconSpread = interpolate(frame, [150, 220], [0, 1]);
  const lingeringFlicker = interpolate(frame, [250, 300], [1, 0.8]);

  return (
    <AbsoluteFill style={{ backgroundColor: '#040c1c', overflow: 'hidden' }}>
      <div style={{ opacity: starsOpacity, position: 'absolute', width: '100%', height: '100%' }}>
        {Array.from({ length: 100 }).map((_, i) => (
          <Star
            key={i}
            size={Math.random() * 5 + 2}
            color="#a0d8ef"
            style={{
              position: 'absolute',
              top: `${Math.random() * 100}%`,
              left: `${Math.random() * 100}%`,
              opacity: Math.random() * 0.8 + 0.2,
            }}
          />
        ))}
      </div>

      <Sequence from={60}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${supernovaScale})`,
            opacity: interpolate(frame, [85, 95], [1, 0]),
          }}
        >
          <div
            style={{
              width: '200px',
              height: '200px',
              borderRadius: '50%',
              background: 'radial-gradient(circle, #ffffff, #00aaff, transparent)',
              boxShadow: '0 0 100px 50px #00aaff',
            }}
          />
        </div>
      </Sequence>

      <Sequence from={90}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${siriusScale}) rotate(${siriusRotation}deg)`,
          }}
        >
          <div
            style={{
              width: '150px',
              height: '150px',
              borderRadius: '50%',
              border: '5px solid #00f0ff',
              background: `radial-gradient(circle, rgba(0, 240, 255, ${siriusCoreOpacity}), rgba(0, 20, 60, 1))`,
              boxShadow: '0 0 50px 20px #00f0ff, inset 0 0 30px 10px #00f0ff',
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
            }}
          >
          </div>
        </div>
      </Sequence>

      <Sequence from={150}>
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${menuScale})`,
            opacity: menuOpacity,
          }}
        >
          <svg
            width="800"
            height="800"
            style={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              opacity: lingeringFlicker,
            }}
          >
            <circle cx="400" cy="400" r={200 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="5,5" />
            <circle cx="400" cy="400" r={300 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="5,5" />
            {[0, 45, 90, 135, 180, 225, 270, 315].map((angle) => {
              const radian = (angle * Math.PI) / 180;
              const x1 = 400 + Math.cos(radian) * 200 * iconSpread;
              const y1 = 400 + Math.sin(radian) * 200 * iconSpread;
              const x2 = 400 + Math.cos(radian) * 300 * iconSpread;
              const y2 = 400 + Math.sin(radian) * 300 * iconSpread;
              return (
                <g key={angle}>
                  
                  <circle cx={x1} cy={y1} r="3" fill="#00f0ff" />
                  <circle cx={x2} cy={y2} r="3" fill="#00f0ff" />
                </g>
              );
            })}
          </svg>
        </div>
      </Sequence>

      
    </AbsoluteFill>
  );
};