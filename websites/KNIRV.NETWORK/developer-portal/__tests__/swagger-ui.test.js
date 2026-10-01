const fs = require('fs');
const path = require('path');

describe('Swagger UI API specifications', () => {
  const portalRoot = path.join(__dirname, '..');
  const swaggerUi = fs.readFileSync(path.join(portalRoot, 'swagger-ui.html'), 'utf8');

  it.each([
    ['Actuarial Syndicate API', 'actuarial-openapi.yaml'],
    ['KNIRV Unified API', 'openapi.yaml'],
  ])('links %s to a deployed OpenAPI document', (_label, specification) => {
    expect(swaggerUi).toContain(`value="${specification}"`);

    const specificationPath = path.join(portalRoot, specification);
    expect(fs.existsSync(specificationPath)).toBe(true);

    const specificationContents = fs.readFileSync(specificationPath, 'utf8');
    expect(specificationContents).toMatch(/^openapi:\s*3\.1\.0\s*$/m);
  });

  it('serves the KNIRV Unified API specification', () => {
    const unifiedSpec = fs.readFileSync(path.join(portalRoot, 'openapi.yaml'), 'utf8');

    expect(unifiedSpec).toContain('title: KNIRV Unified API');
    expect(unifiedSpec).toContain('/payment/api/faucet/request');
    expect(unifiedSpec).toContain('/operator-registry/register');
  });
});
