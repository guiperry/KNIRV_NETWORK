```javascript
import { AbsoluteFill, interpolate, useCurrentFrame, Sequence } from 'remotion';
import { Star, Zap } from 'lucide-react'; // Using Lucide icons for effect, replace with actual image assets

// Use this prompt in a Remotion project to animate the provided image.
// Ensure you have the necessary assets separated from the image_0.png for each component.

export const KnirvanaAnimation = () => {
  const frame = useCurrentFrame();
  const duration = 300; // Total duration in frames

  // Phase 1: Stars appearing (0-60 frames)
  const starsOpacity = interpolate(frame, [0, 60], [0, 1]);

  // Phase 2: Supernova Explosion & Sirius Star (60-150 frames)
  const supernovaScale = interpolate(frame, [60, 90], [0.1, 3]);
  const siriusScale = interpolate(frame, [90, 120, 150], [0, 1.2, 0.8]);
  const siriusRotation = interpolate(frame, [90, 150], [0, 360]);
  const siriusCoreOpacity = interpolate(frame, [120, 150], [1, 0.3]); // Darken center
  const electricSparksOpacity = interpolate(frame, [120, 150], [0, 1]);

  // Phase 3: Menu Opening (150-250 frames)
  const menuScale = interpolate(frame, [150, 220], [0, 1]);
  const menuOpacity = interpolate(frame, [150, 180], [0, 1]);
  const iconSpread = interpolate(frame, [150, 220], [0, 1]); // Control icon positions along lines

  // Phase 4: Lingering Effects (250-300 frames)
  const lingeringFlicker = interpolate(frame, [250, 300], [1, 0.8]); // Subtle flicker

  return (
    <AbsoluteFill style={{ backgroundColor: '#040c1c', overflow: 'hidden' }}>
      {/* Stars Background */}
      <div style={{ opacity: starsOpacity, position: 'absolute', width: '100%', height: '100%' }}>
        {/* Generate many small stars here */}
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
        {/* Supernova */}
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${supernovaScale})`,
            opacity: interpolate(frame, [85, 95], [1, 0]), // Fade out quickly
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
        {/* Sirius Star & Electric Core */}
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
            {/* Electric Spires */}
            <div style={{ opacity: electricSparksOpacity }}>
              <Zap color="#00f0ff" size={40} style={{ position: 'absolute', top: '-20px', left: '50%', transform: 'translateX(-50%)' }} />
              <Zap color="#00f0ff" size={40} style={{ position: 'absolute', bottom: '-20px', left: '50%', transform: 'translateX(-50%) rotate(180deg)' }} />
              {/* Add more random sparks */}
            </div>
          </div>
        </div>
      </Sequence>

      <Sequence from={150}>
        {/* KNIRVANA Menu System */}
        <div
          style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: `translate(-50%, -50%) scale(${menuScale})`,
            opacity: menuOpacity,
          }}
        >
          {/* Central Text */}
          <h1 style={{ color: '#ffffff', fontSize: '3em', textAlign: 'center', textShadow: '0 0 20px #00f0ff' }}>KNIRVANA</h1>

          {/* Inner Ring Icons & Text */}
          <div style={{ position: 'absolute', top: '50%', left: '50%', width: '400px', height: '400px', transform: 'translate(-50%, -50%)' }}>
            <div style={{ position: 'absolute', top: 0, left: '50%', transform: `translate(-50%, ${-iconSpread * 180}px)` }}>KNIRV PAY</div>
            <div style={{ position: 'absolute', bottom: 0, left: '50%', transform: `translate(-50%, ${iconSpread * 180}px)` }}>KNIRV CHAIN</div>
            {/* ... Add other inner ring elements ... */}
          </div>

          {/* Outer Ring Icons & Text */}
          <div style={{ position: 'absolute', top: '50%', left: '50%', width: '600px', height: '600px', transform: 'translate(-50%, -50%)' }}>
            <div style={{ position: 'absolute', top: 0, left: '50%', transform: `translate(-50%, ${-iconSpread * 280}px)` }}>
              <span style={{ fontSize: '2em' }}>V</span> <br /> KNIRV GATEWAY
            </div>
            {/* ... Add other outer ring elements ... */}
          </div>

          {/* Constellation Lines & Dots */}
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
            {/* Draw lines and dots connecting the components with electric effect */}
            <circle cx="400" cy="400" r={200 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="5,5" />
            <circle cx="400" cy="400" r={300 * iconSpread} stroke="#00f0ff" strokeWidth="2" fill="none" strokeDasharray="5,5" />
            {/* ... Add more complex lines and dots with electric effects ... */}
          </svg>
        </div>
      </Sequence>

      {/* Bottom Text */}
      <div style={{ position: 'absolute', bottom: '30px', width: '100%', textAlign: 'center', color: '#00f0ff', fontSize: '2em', opacity: menuOpacity }}>
        KNIRV.COM
      </div>
    </AbsoluteFill>
  );
};

```



Generate an interactive game menu in TypeScript that looks exactly like the attached picture.

---

**Title:** Sci-Fi UI Reveal: The KNIRVANA Constellation

**Composition Settings:**

* **Width:** 1080px
* **Height:** 1080px (Square format matching the image)
* **FPS:** 60 (for smooth particle and rotation effects)
* **Duration:** Approximately 12-15 seconds

**Assets Required (derived from orb_2.png):**

1. **Background Layer:** The dark, starry night sky and the blue-lit crystal landscape at the horizon.
2. **UI Elements (separated into individual transparent PNGs):**
* Central "KNIRVANA" text sphere.
* Inner Rings (KNIRV PAY, KNIRV CHAIN, etc.).
* Middle Rings (KNIRV CORTEX, KNIRV CONTROLLER, etc.).
* Outer Rings & Connecting Lines (The "constellation lines").
* Outer Icons (The 8 circular icons: Cube, Eye, 'X', Down Arrow, Wrench/Gear, Wrench, Globe, 'V' logo).


3. **Text Element:** "KNIRV.COM" logo.
4. **Effects Assets:**
* Particle system for stars.
* A bright, rotating "Sirius Star" asset with an electric blue corona.
* Electric lightning bolt/spire sprites.



**Animation Sequence Prompt:**

**Phase 1: The Cosmos Awakens (0:00 - 0:03)**

* **Initial State:** The screen is completely black.
* **Action:** Slowly fade in the background layer (stars and crystal landscape). Simultaneously, generate a particle system of small, glowing blue and white stars across the frame.
* **Transition:** The particles begin to drift, then accelerate, swirling chaotically toward the exact dead center of the screen, forming a dense, bright point of light.

**Phase 2: The Supernova Event (0:03 - 0:05)**

* **Action:** The central point of concentrated stars detonates into a massive supernova. This should be a blinding flash of white and cyan light that rapidly expands to cover nearly the entire screen, briefly obscuring the background.
* **Dissipation:** As the flash subsides, it reveals a intensely bright, burning blue "Sirius Star" in the center.

**Phase 3: The Star Stabilizes (0:05 - 0:07)**

* **Sirius Star Action:** The central blue star rotates rapidly. As it spins, it shrinks slightly in diameter.
* **Color Shift:** The core of the star transitions from blinding white/blue to the deeper, darker blue energy sphere seen in the center of `image_0.png`.
* **Edge Effect:** The outer edge of this central sphere remains intensely bright electric blue. Random, jagged electric spires and sparks must erupt from this edge, crackling outwards into the surrounding space.

**Phase 4: The UI Constellation Unfolds (0:07 - 0:11)**

* **The Reveal:** The "KNIRVANA" text fades into the dark center of the sphere.
* **Expansion:** The entire complex circular menu system (rings, lines, text, and icons from `image_0.png`) emerges from the center point of the star.
* **Motion:**
* The concentric rings should scale up from 0% to 100% with an elastic bounce effect.
* The connecting "constellation lines" must draw themselves outwards from the center, looking like flowing electric current (using stroke-dashoffset animation).
* The text labels and outer icons do not just fade in; they must travel along these expanding electric lines from the center to their final positions shown in the reference image.



**Phase 5: Lingering Energy (0:11 - 0:15)**

* **Final State:** The full UI from the attached `orb_2.png` is established.
* **Idle Animation:**
* The entire UI structure should have a very slow, subtle constant rotation (e.g., 1 degree every few seconds).
* The electric spires at the center continue to spark randomly.
* Apply a subtle brightness flicker and glow animation to all the blue lines, text, and icons, simulating live electricity pulsing through the constellation.


* **Closing:** The "KNIRV.COM" text at the bottom fades in smoothly.





This prompt is engineered to provide an AI (like **v0.dev** or **Claude 3.5 Sonnet**) with the technical "blueprint" it needs to replicate the geometry, glow, and physics of that menu.

Copy and paste the block below into your chosen AI:

---

### The "KNIRVANA" System Prompt

> **Task:** Create a high-fidelity, interactive radial "Constellation Menu" in **React**, **Tailwind CSS**, and **Framer Motion** based on the attached image.
> **1. Core Geometry & Layout:**
> * **Central Hub:** A glowing dark-blue sphere with the text "KNIRVANA" in a clean, futuristic sans-serif font.
> * **Tier 1 (Inner Ring):** Labels like "KNIRV PAY" and "KNIRV CHAIN" positioned at 0°, 90°, 180°, and 270°.
> * **Tier 2 (Middle Ring):** Labels including "KNIRV CORTEX" and "KNIRV CONTROLLER" on a wider radius.
> * **Tier 3 (Outer Icons):** 8 circular nodes containing distinct SVG icons (Eye, Cube, 'X', Arrow, Globe, etc.).
> * **Constellation Lines:** Use an SVG overlay to draw thin, glowing cyan lines connecting the central hub to the Tier 1 labels, Tier 2 labels, and finally the Tier 3 icons. Each line should end in a small circular "dot" (vertex).
> 
> 
> **2. Visual Aesthetic:**
> * **Palette:** Deep Navy (#040c1c) background; Electric Cyan (#00f0ff) for lines and text; White/Silver for highlights.
> * **Glow/Bloom:** Apply `drop-shadow` and `box-shadow` to all lines and icons to simulate a neon "HUD" glow.
> * **Glassmorphism:** The outer icons should have a semi-transparent, frosted-glass background.
> 
> 
> **3. Interaction & Animation:**
> * **Entrance:** On mount, use Framer Motion to animate the center star expanding. The constellation lines should use `pathLength` animation to "draw" themselves outward, and icons should "pop" into place once their line reaches them.
> * **Hover States:** When hovering over an icon, it should scale up (e.g., `scale: 1.2`), its glow should intensify, and the line connecting it to the center should "pulse" with brightness.
> * **Idle Motion:** Implement a very subtle "breathing" effect where the entire menu slowly expands and contracts by 1-2%.
> 
> 
> **4. Technical Constraint:**
> * Use **Polar Coordinates** (`sin` and `cos` in CSS/JS) to ensure the nodes are perfectly spaced in a circle. Ensure the layout is responsive and centers perfectly on the screen.
> 
> 

---

### Why this prompt works "The Best":

* **Polar Coordinates:** Most AI models try to use absolute positioning (top/left), which breaks easily. Telling it to use math (`sin/cos`) ensures the circles are perfect.
* **PathLength Animation:** This is the secret to the "drawing" effect you see in high-end game menus.
* **Framer Motion:** It’s the industry standard for React animations; it handles the physics (the "spring" and "bounce") better than any other tool.

### How to use this:

1. **Go to [v0.dev](https://v0.dev).**
2. **Upload the original image** you gave me.
3. **Paste the prompt above** into the message box.
4. **Review the code:** v0 will give you a "Copy Code" button. You can then drop that straight into a React project.

**Would you like me to break down the specific math (Sine/Cosine) you'll need if you decide to code the positions by hand?**