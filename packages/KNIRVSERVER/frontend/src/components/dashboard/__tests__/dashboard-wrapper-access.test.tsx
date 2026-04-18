import fs from 'fs';
import path from 'path';

describe('DashboardWrapper DVE access flow', () => {
  it('opens the workspace modal inside handleNodeAccess', () => {
    const filePath = path.join(__dirname, '..', 'dashboard-wrapper.tsx');
    const source = fs.readFileSync(filePath, 'utf8');

    expect(source).toMatch(
      /const handleNodeAccess = \(node: DVENode\) => \{[\s\S]*setSelectedNode\(node\);[\s\S]*setUseModularCDE\(true\);[\s\S]*setCdeModalOpen\(true\);[\s\S]*\};/
    );
  });
});
