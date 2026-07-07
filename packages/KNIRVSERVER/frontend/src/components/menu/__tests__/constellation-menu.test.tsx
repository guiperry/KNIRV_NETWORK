import fs from 'fs';
import path from 'path';

describe('ConstellationMenu navigation flow', () => {
  it('handleIconClick routes arena section directly to /arena', () => {
    const filePath = path.join(__dirname, '..', 'constellation-menu.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    const handleIconClickMatch = source.match(
      /const handleIconClick = \(section: string \| null\) => \{[\s\S]*?\n  }/m
    );
    expect(handleIconClickMatch).not.toBeNull();

    const handler = handleIconClickMatch![0];
    expect(handler).toMatch(/section === 'arena'/);
    expect(handler).toMatch(/router\.push\('\/arena'\)/);
  });

  it('center orb navigates to /arena on zoom completion', () => {
    const filePath = path.join(__dirname, '..', 'constellation-menu.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    expect(source).toMatch(/router\.push\('\/arena'\)/);
    expect(source).toMatch(/window\.parent\.postMessage\(\{ type: 'navigate', section: 'arena' \}, '\*'\)/);
  });
});
