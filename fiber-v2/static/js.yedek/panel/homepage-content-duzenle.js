/**
 * Homepage Content Duzenle Page JavaScript
 * Mirrors interactivity from Custom Content edit page, with CodeMirror editors
 */

class HomepageContentEditHandler {
    constructor() {
        this.hcid = window.pageData?.hcid;
        this.form = document.getElementById('homepageContentEditForm');
        this.submitBtn = document.getElementById('submitBtn');

        this.cmHtml = null;
        this.cmCss = null;
        this.cmJs = null;

        this.init();
    }

    init() {
        this.initializeCodeEditors();
        this.setupForm();
        this.setupToggleSwitches();
    }

    initializeCodeEditors() {
        const htmlTextarea = document.getElementById('content_html');
        const cssTextarea = document.getElementById('content_css');
        const jsTextarea = document.getElementById('content_javascript');

        if (htmlTextarea) {
            this.cmHtml = CodeMirror.fromTextArea(htmlTextarea, {
                mode: 'htmlmixed',
                lineNumbers: true,
                theme: 'monokai',
                matchBrackets: true,
                autoCloseBrackets: true,
                extraKeys: { 'Ctrl-Space': 'autocomplete' }
            });
        }

        if (cssTextarea) {
            this.cmCss = CodeMirror.fromTextArea(cssTextarea, {
                mode: 'css',
                lineNumbers: true,
                theme: 'monokai',
                matchBrackets: true,
                autoCloseBrackets: true,
                extraKeys: { 'Ctrl-Space': 'autocomplete' }
            });
        }

        if (jsTextarea) {
            this.cmJs = CodeMirror.fromTextArea(jsTextarea, {
                mode: 'javascript',
                lineNumbers: true,
                theme: 'monokai',
                matchBrackets: true,
                autoCloseBrackets: true,
                extraKeys: { 'Ctrl-Space': 'autocomplete' }
            });
        }
    }

    setupForm() {
        if (!this.form) return;
        this.form.addEventListener('submit', async (e) => {
            e.preventDefault();

            // Sync CodeMirror contents back to textareas
            if (this.cmHtml) this.cmHtml.save();
            if (this.cmCss) this.cmCss.save();
            if (this.cmJs) this.cmJs.save();

            const formData = new FormData(this.form);

            this.setLoadingState(true);
            try {
                const response = await fetch(`/backend/homepage-content/${this.hcid}/edit`, {
                    method: 'POST',
                    body: formData
                });

                const result = await response.json();
                if (result.status === 201) {
                    this.showSuccessModal('Anasayfa içeriği başarıyla güncellendi.');
                } else {
                    throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
                }
            } catch (error) {
                console.error('Homepage content update error:', error);
                this.showErrorModal(error.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            } finally {
                this.setLoadingState(false);
            }
        });
    }

    setupToggleSwitches() {
        const toggles = document.querySelectorAll('.toggle-switch input[type="checkbox"]');
        toggles.forEach(toggle => {
            const updateToggleText = () => {
                const textElement = toggle.parentNode.querySelector('.toggle-text');
                if (textElement) {
                    textElement.textContent = toggle.checked ? 'Aktif' : 'Pasif';
                }
            };
            toggle.addEventListener('change', updateToggleText);
            updateToggleText();
        });
    }

    setLoadingState(loading) {
        if (!this.submitBtn || !this.form) return;
        if (loading) {
            this.submitBtn.disabled = true;
            this.submitBtn.querySelector('.btn-text').style.opacity = '0';
            this.submitBtn.querySelector('.btn-loader').style.display = 'block';
            this.form.classList.add('form-loading');
        } else {
            this.submitBtn.disabled = false;
            this.submitBtn.querySelector('.btn-text').style.opacity = '1';
            this.submitBtn.querySelector('.btn-loader').style.display = 'none';
            this.form.classList.remove('form-loading');
        }
    }

    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        if (messageElement) messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        if (messageElement) messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new HomepageContentEditHandler();

    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            const modalId = e.target.id;
            const modal = document.getElementById(modalId);
            modal.classList.remove('show');
            setTimeout(() => { modal.style.display = 'none'; }, 300);
        }
    });

    // Close modals with Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.show');
            if (visibleModal) {
                visibleModal.classList.remove('show');
                setTimeout(() => { visibleModal.style.display = 'none'; }, 300);
            }
        }
    });
});


document.addEventListener('DOMContentLoaded', () => {
    const oldBannerInput = document.getElementById('bannerFileInput');
    if (!oldBannerInput) return;

    const bannerInput = oldBannerInput.cloneNode(true);
    oldBannerInput.parentNode.replaceChild(bannerInput, oldBannerInput);

    bannerInput.addEventListener('change', async function (e) {
        const file = e.target.files[0];
        if (!file) return;

        const formData = new FormData();
        formData.append('file', file);

        try {
            const response = await fetch('/backend/add-file', {
                method: 'POST',
                body: formData,
                credentials: 'same-origin',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const text = await response.text();
            let data;

            try {
                data = JSON.parse(text);
            } catch (err) {
                throw new Error('Sunucu JSON yerine şunu döndürdü: ' + text.slice(0, 200));
            }

            if ((data.status === 200 || data.status === 201) && data.data && data.data[0]) {
                const rawPath = data.data[0].Url || data.data[0].url || '';
                const path = rawPath
                    .replace(/\\/g, '/')
                    .replace(/^.*\/static\//, '')
                    .replace(/^\/+/, '');

                const htmlTextarea = document.getElementById('content_html');
                if (htmlTextarea) {
                    htmlTextarea.value = path;
                }

                const cmElement = document.querySelector('#content_html + .CodeMirror') || document.querySelector('.CodeMirror');
                if (cmElement && cmElement.CodeMirror) {
                    cmElement.CodeMirror.setValue(path);
                }

                const currentImgBlock = document.getElementById('bannerCurrentImg');
                if (currentImgBlock) {
                    currentImgBlock.style.display = 'none';
                }

                const previewImg = document.getElementById('bannerPreviewImg');
                const previewPath = document.getElementById('bannerPreviewPath');
                const previewWrap = document.getElementById('bannerPreview');
                const uploadArea = document.getElementById('bannerUploadArea');

                if (previewImg) previewImg.src = '/' + path;
                if (previewPath) previewPath.textContent = path;
                if (previewWrap) previewWrap.style.display = 'block';
                if (uploadArea) uploadArea.style.borderColor = '#10b981';

                alert('Görsel başarıyla yüklendi.');
            } else {
                throw new Error(data.message || 'Bilinmeyen hata');
            }
        } catch (err) {
            alert('Yükleme hatası: ' + err.message);
            console.error('Banner upload error:', err);
        }
    });
});

document.addEventListener('DOMContentLoaded', () => {
    const oldMobileBannerInput = document.getElementById('mobileBannerFileInput');
    if (!oldMobileBannerInput) return;

    const mobileBannerInput = oldMobileBannerInput.cloneNode(true);
    oldMobileBannerInput.parentNode.replaceChild(mobileBannerInput, oldMobileBannerInput);

    mobileBannerInput.addEventListener('change', async function (e) {
        const file = e.target.files[0];
        if (!file) return;

        const formData = new FormData();
        formData.append('file', file);

        try {
            const response = await fetch('/backend/add-file', {
                method: 'POST',
                body: formData,
                credentials: 'same-origin',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const text = await response.text();
            let data;

            try {
                data = JSON.parse(text);
            } catch (err) {
                throw new Error('Sunucu JSON yerine şunu döndürdü: ' + text.slice(0, 200));
            }

            if ((data.status === 200 || data.status === 201) && data.data && data.data[0]) {
                const rawPath = data.data[0].Url || data.data[0].url || '';
                const path = rawPath
                    .replace(/\\/g, '/')
                    .replace(/^.*\/static\//, '')
                    .replace(/^\/+/, '');

                const cssTextarea = document.getElementById('content_css');
                if (cssTextarea) {
                    cssTextarea.value = path;
                }

                const cmElements = document.querySelectorAll('.CodeMirror');
                if (cmElements.length > 1 && cmElements[1].CodeMirror) {
                    cmElements[1].CodeMirror.setValue(path);
                }

                const currentImgBlock = document.getElementById('mobileBannerCurrentImg');
                if (currentImgBlock) {
                    currentImgBlock.style.display = 'none';
                }

                const previewImg = document.getElementById('mobileBannerPreviewImg');
                const previewPath = document.getElementById('mobileBannerPreviewPath');
                const previewWrap = document.getElementById('mobileBannerPreview');
                const uploadArea = document.getElementById('mobileBannerUploadArea');

                if (previewImg) previewImg.src = '/' + path;
                if (previewPath) previewPath.textContent = path;
                if (previewWrap) previewWrap.style.display = 'block';
                if (uploadArea) uploadArea.style.borderColor = '#10b981';

                alert('Mobil görsel başarıyla yüklendi.');
            } else {
                throw new Error(data.message || 'Bilinmeyen hata');
            }
        } catch (err) {
            alert('Mobil görsel yükleme hatası: ' + err.message);
            console.error('Mobile banner upload error:', err);
        }
    });
});
