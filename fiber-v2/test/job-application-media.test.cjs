const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const source = fs.readFileSync(
    path.join(__dirname, '../static/js/panel/job-application.js'),
    'utf8'
);

function managerWithDOM(previewButtons = []) {
    const elements = {
        documentPreviewModal: { style: {}, classList: { add() {} } },
        documentPreviewContainer: {
            innerHTML: '',
            children: [],
            appendChild(child) { this.children.push(child); }
        },
        documentPreviewTitle: { textContent: '' },
        documentPreviewInfo: { textContent: '' }
    };
    const clicks = [];
    const document = {
        addEventListener() {},
        getElementById(id) { return elements[id]; },
        querySelectorAll() { return previewButtons; },
        createElement(tag) {
            return { tag, style: {}, click() { clicks.push(this.href); } };
        },
        body: {
            style: {},
            appendChild() {},
            removeChild() {}
        }
    };
    const context = vm.createContext({
        document,
        window: {},
        setTimeout(callback) { callback(); }
    });
    vm.runInContext(source, context);
    const manager = vm.runInContext(
        'Object.create(JobApplicationViewManager.prototype)',
        context
    );
    return { manager, elements, clicks };
}

test('PDF preview and download use the protected media URL', () => {
    const { manager, elements, clicks } = managerWithDOM();
    const url = '/panel/is-basvurulari/7/media/10?preview=1';

    manager.openDocumentPreview(url, 'CV Dosyası', '.pdf');

    assert.equal(elements.documentPreviewContainer.children[0].src, url);
    assert.equal(elements.documentPreviewInfo.textContent, 'CV Dosyası');
    assert.equal(elements.documentPreviewModal.style.display, 'flex');

    manager.downloadDocument();
    assert.deepEqual(clicks, ['/panel/is-basvurulari/7/media/10']);
});

test('preview button passes the extension separately from its URL', () => {
    let click;
    let args;
    const button = {
        addEventListener(_event, handler) { click = handler; },
        getAttribute(name) {
            return {
                'data-filepath': '/panel/is-basvurulari/7/media/11?preview=1',
                'data-filetype': '.docx',
                'data-filename': 'Diploma Dosyası'
            }[name];
        }
    };
    const { manager } = managerWithDOM([button]);
    manager.openDocumentPreview = (...values) => { args = values; };

    manager.setupDocumentPreview();
    click({ preventDefault() {} });

    assert.deepEqual(args, [
        '/panel/is-basvurulari/7/media/11?preview=1',
        'Diploma Dosyası',
        '.docx'
    ]);
});
