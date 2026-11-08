const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');
const vm = require('node:vm');
const test = require('node:test');

// Load the production predicate without needing a browser or Angular runtime.
const source = fs.readFileSync(path.join(__dirname, '../src/app/shared/local-settings.ts'), 'utf8');
const context = { exports: {} };
vm.runInNewContext(ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText, context);
const { isLocalSetting } = context.exports;

test('global and profile settings exclude every SPN key and upstream binary update controls', () => {
  for (const Key of ['spn/enable', 'spn/use', 'spn/exitHubPolicy', 'spn/futureOption', 'core/releaseChannel', 'core/automaticUpdates']) {
    assert.equal(isLocalSetting({ Key }), false, Key);
  }
});

test('local features and intelligence update settings remain available', () => {
  for (const Key of ['filter/endpoints', 'filter/defaultAction', 'history/enable', 'netquery/enable', 'core/automaticIntelUpdates', 'core/releaseLevel', 'splittun/use']) {
    assert.equal(isLocalSetting({ Key }), true, Key);
  }
});
