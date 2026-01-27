import { RFTAgent } from '../types/Agent';

export enum SabotageType {
    NOISE_INJECTION = 'NOISE_INJECTION', // Adds random chars to context
    BACKPROP_PULSE = 'BACKPROP_PULSE',   // Reduces target Parity
    GRADIENT_GHOSTING = 'GRADIENT_GHOSTING' // Creates fake high-reward target
}

export class SabotageEngine {
    
    public static applyEffect(type: SabotageType, target: RFTAgent, magnitude: number) {
        switch (type) {
            case SabotageType.NOISE_INJECTION:
                // Logic: Return a "Decorator" function that corrupts the input string
                // for the target's next Inference Step.
                console.log(`Noise injection applied to ${target.id} with magnitude ${magnitude}`);
                break;
                
            case SabotageType.BACKPROP_PULSE:
                // Logic: Directly reduce Parity
                target.resources.parity -= (10 * magnitude);
                console.log(`Backprop pulse applied to ${target.id}. Parity reduced by ${10 * magnitude}`);
                break;
                
            case SabotageType.GRADIENT_GHOSTING:
                // Logic: Create fake high-reward target
                console.log(`Gradient ghosting applied to ${target.id} with magnitude ${magnitude}`);
                break;
        }
    }
}
