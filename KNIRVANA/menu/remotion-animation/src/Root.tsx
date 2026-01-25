import { Composition } from 'remotion';
import { KnirvanaAnimation } from './KnirvanaAnimation';
import KnirvanaInteractiveMenu from './KnirvanaInteractiveMenu';

export const RemotionRoot: React.FC = () => {
	return (
		<>
			<Composition
				id="Knirvana"
				component={KnirvanaAnimation}
				durationInFrames={300}
				fps={30}
				width={1920}
				height={1080}
			/>
			<Composition
				id="KnirvanaInteractive"
				component={KnirvanaInteractiveMenu}
				durationInFrames={300}
				fps={30}
				width={1920}
				height={1080}
			/>
		</>
	);
};