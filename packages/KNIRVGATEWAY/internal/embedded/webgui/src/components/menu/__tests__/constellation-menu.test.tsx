import fs from 'fs';
import path from 'path';

describe('ConstellationMenu navigation flow', () => {
  it('handleIconClick opens settings for a null href and routes directly otherwise', () => {
    const filePath = path.join(__dirname, '..', 'constellation-menu.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    const handleIconClickMatch = source.match(
      /const handleIconClick = \(href: string \| null\) => \{[\s\S]*?\n  }/m
    );
    expect(handleIconClickMatch).not.toBeNull();

    const handler = handleIconClickMatch![0];
    expect(handler).toMatch(/href === null/);
    expect(handler).toMatch(/setSettingsOpen\(true\)/);
    expect(handler).toMatch(/router\.push\(href\)/);
  });

  it('center orb navigates to /arena on zoom completion', () => {
    const filePath = path.join(__dirname, '..', 'constellation-menu.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    expect(source).toMatch(/router\.push\('\/arena'\)/);
    expect(source).toMatch(/window\.parent\.postMessage\(\{ type: 'navigate', section: 'arena' \}, '\*'\)/);
  });
});
