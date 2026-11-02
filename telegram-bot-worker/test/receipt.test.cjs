const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

const image = Buffer.from([0xff, 0xd8, 0xff, 0xe0, 0x01, 0xff, 0xd9]);
const receipt = `data:image/jpeg;base64,${image.toString('base64')}`;
const env = {
  BACKEND_URL: 'https://backend.test', TELEGRAM_BOT_TOKEN: 'test-token',
  TELEGRAM_BOT_SECRET: 'test-secret', GEMINI_API_KEY: 'test-key',
};
const photo = [{ file_id: 'small', width: 10, height: 10, file_unique_id: 's' },
  { file_id: 'original-slip', width: 100, height: 100, file_unique_id: 'l' }];

function replay({ downloadFails = false } = {}) {
  const calls = [];
  const fetch = async (input, options = {}) => {
    const url = String(input);
    const body = options.body ? JSON.parse(options.body) : undefined;
    calls.push({ url, body });
    const json = (value) => Response.json(value);
    if (url.includes('/getFile?')) return json({ ok: true, result: { file_path: 'photos/slip.jpg' } });
    if (url.includes('/file/bot')) return downloadFails
      ? new Response('unavailable', { status: 503 })
      : new Response(image, { headers: { 'Content-Type': 'image/jpeg' } });
    if (url.includes('generativelanguage.googleapis.com')) return json({ candidates: [{ content: { parts: [{ text: JSON.stringify({ amount: 450, category: 'Utilities', description: 'electricity' }) }] } }] });
    if (url.includes('/pending-bills')) return json({ pendingBills: [{ id: 'bill-1', amount: 450, description: 'electricity' }] });
    if (url.includes('/api/telegram/pay-bill') || url.endsWith('/api/telegram/transactions')) return json({ id: 'saved', amount: 450, paidAmount: 450, receiptImage: body.receiptImage });
    return json({ ok: true, result: { message_id: 200 } });
  };
  const source = fs.readFileSync(require('node:path').join(__dirname, '../src/index.ts'), 'utf8');
  const compiled = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 } }).outputText;
  const sandbox = { exports: {}, fetch, Request, Response, URL, Uint8Array, btoa, console };
  vm.runInNewContext(compiled, sandbox);
  const dispatch = async (update) => sandbox.exports.default.fetch(new Request('https://worker.test', {
    method: 'POST', body: JSON.stringify(update), headers: { 'Content-Type': 'application/json' },
  }), env);
  return { calls, dispatch };
}

for (const action of ['pay:bill-1', 'new:450:2']) {
  test(`${action}: photo confirmation survives a new Worker instance and saves a viewable receipt`, async () => {
    const first = replay();
    const original = { message_id: 42, chat: { id: -123, type: 'group' }, from: { id: 7, first_name: 'User' }, date: 1, photo };
    await first.dispatch({ update_id: 1, message: original });
    const confirmation = first.calls.find((c) => c.url.endsWith('/sendMessage') && c.body.reply_markup?.inline_keyboard);
    assert.ok(confirmation, 'slip produces confirmation buttons');
    assert.equal(confirmation.body.reply_parameters?.message_id, original.message_id, 'confirmation must retain its source photo reference');
    assert.ok(confirmation.body.reply_markup.inline_keyboard.flat().some((b) => b.callback_data === action));
    const second = replay();
    await second.dispatch({ update_id: 2, callback_query: {
      id: 'cb', from: original.from, data: action,
      message: { message_id: 200, chat: original.chat, date: 1, reply_to_message: original },
    } });
    const saved = second.calls.find((c) => c.url.includes('/api/telegram/pay-bill') || c.url.endsWith('/api/telegram/transactions'));
    assert.ok(saved, 'confirmation writes payment');
    assert.equal(saved.body.receiptImage, receipt);
    assert.deepEqual(Buffer.from(saved.body.receiptImage.split(',')[1], 'base64'), image);
    assert.ok(second.calls.some((c) => c.url.includes('/getFile?file_id=original-slip')), 'fetches highest resolution source');
  });

  for (const failure of ['missing source', 'download failed']) {
    test(`${action}: ${failure} does not save a paid transaction without a receipt`, async () => {
      const run = replay({ downloadFails: failure === 'download failed' });
      await run.dispatch({ update_id: 3, callback_query: {
        id: 'cb', from: { id: 7, first_name: 'User' }, data: action,
        message: { message_id: 200, chat: { id: -123, type: 'group' }, date: 1,
          ...(failure === 'download failed' ? { reply_to_message: { photo } } : {}) },
      } });
      assert.equal(run.calls.filter((c) => c.url.includes('/api/telegram/pay-bill') || c.url.endsWith('/api/telegram/transactions')).length, 0);
      assert.ok(run.calls.some((c) => c.url.endsWith('/editMessageText') && c.body.text.startsWith('❌')));
    });
  }
}

test('cancel does not download or save a slip', async () => {
  const run = replay();
  await run.dispatch({ callback_query: { id: 'cb', from: { id: 7, first_name: 'User' }, data: 'cancel',
    message: { message_id: 200, chat: { id: -123, type: 'group' }, date: 1 } } });
  assert.equal(run.calls.some((c) => c.url.includes('/getFile') || c.url.includes('/api/telegram/')), false);
});
