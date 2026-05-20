import fs from 'fs';
import path from 'path';

describe('DashboardWrapper DVE access flow', () => {
  it('opens the workspace modal inside handleNodeAccess', () => {
    const filePath = path.join(__dirname, '..', 'dashboard-wrapper.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    expect(source).toMatch(
      /const handleNodeAccess = \(node: DVENode\) => \{[^}]*setSelectedNode\(node\);[^}]*setCdeModalOpen\(true\);[^}]*};/
    );
  });

  it('KNIRVARENA Card navigates to /arena, not admin access', () => {
    const filePath = path.join(__dirname, '..', 'dashboard-wrapper.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    const arenaCardMatch = source.match(/KNIRVARENA[\s\S]*?<\/Card>/);
    expect(arenaCardMatch).not.toBeNull();

    const arenaCardBlock = arenaCardMatch![0];
    expect(arenaCardBlock).toContain("router.push('/arena')");
    expect(arenaCardBlock).not.toContain('setShowAdminAccess');
  });
});
