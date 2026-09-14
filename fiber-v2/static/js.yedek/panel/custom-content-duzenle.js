/**
 * Custom Content Duzenle Page JavaScript
 * + Ulasim mapping fix (ulasim -> other, url_name/route defaults)
 * + credentials: same-origin
 */

class CustomContentEditHandler {
    constructor() {
        this.ccid = window.pageData?.ccid;
        this.form = document.getElementById('customContentEditForm');
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

        if (htmlTextarea && window.CodeMirror) {
            this.cmHtml = CodeMirror.fromTextArea(htmlTextarea, {
                mode: 'htmlmixed',
                lineNumbers: true,
                theme: 'monokai',
                matchBrackets: true,
                autoCloseBrackets: true,
                extraKeys: { 'Ctrl-Space': 'autocomplete' }
            });
        }

        if (cssTextarea && window.CodeMirror) {
            this.cmCss = CodeMirror.fromTextArea(cssTextarea, {
                mode: 'css',
                lineNumbers: true,
                theme: 'monokai',
                matchBrackets: true,
                autoCloseBrackets: true,
                extraKeys: { 'Ctrl-Space': 'autocomplete' }
            });
        }

        if (jsTextarea && window.CodeMirror) {
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

    /**
     * UI "ulasim" -> DB "other"
     * Also ensure url_name + route for Ulasim
     */
    applyUlasimMapping(formData) {
        const ctEl = document.getElementById('content_type');
        const urlEl = document.getElementById('url_name');
        const routeEl = document.getElementById('route');

        if (!ctEl) return;

        if (ctEl.value === 'ulasim') {
            // Ensure fields exist (frontend currently reads url_name=ulasim)
            if (urlEl && (!urlEl.value || urlEl.value.trim() === '')) urlEl.value = 'ulasim';
            if (routeEl && (!routeEl.value || routeEl.value.trim() === '')) routeEl.value = '/ulasim';

            // Update FormData payload
            formData.set('url_name', 'ulasim');
            formData.set('route', '/ulasim');

            // IMPORTANT: send DB-safe type
            // // formData.set('content_type', 'other');
        }
    }

    /**
     * Ensure checkbox "is_active" always arrives deterministically.
     * (If unchecked, HTML form may omit it.)
     */
    normalizeIsActive(formData) {
        const isActiveEl = document.getElementById('is_active');
        if (!isActiveEl) return;

        // Overwrite with explicit value
        if (isActiveEl.checked) {
            formData.set('is_active', 'on');
        } else {
            // some parsers treat missing as false, but we force it:
            formData.set('is_active', 'false');
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

            // Fix checkbox + Ulasim mapping
            this.normalizeIsActive(formData);
            this.applyUlasimMapping(formData);

            this.setLoadingState(true);
            try {
                const response = await fetch(`/backend/custom-content/${this.ccid}/edit`, {
                    method: 'POST',
                    body: formData,
                    credentials: 'same-origin'
                });

                // Backend always returns JSON
                const result = await response.json();

                if (result.status === 201) {
                    this.showSuccessModal('Özel içerik başarıyla güncellendi.');
                } else {
                    throw new Error(result.message || 'Güncelleme işlemi başarısız oldu.');
                }
            } catch (error) {
                console.error('Custom content update error:', error);
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

        const btnText = this.submitBtn.querySelector('.btn-text');
        const btnLoader = this.submitBtn.querySelector('.btn-loader');

        if (loading) {
            this.submitBtn.disabled = true;
            if (btnText) btnText.style.opacity = '0';
            if (btnLoader) btnLoader.style.display = 'block';
            this.form.classList.add('form-loading');
        } else {
            this.submitBtn.disabled = false;
            if (btnText) btnText.style.opacity = '1';
            if (btnLoader) btnLoader.style.display = 'none';
            this.form.classList.remove('form-loading');
        }
    }

    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        if (messageElement) messageElement.textContent = message;
        if (!modal) return;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        if (messageElement) messageElement.textContent = message;
        if (!modal) return;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new CustomContentEditHandler();

    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
        if (e.target.classList && e.target.classList.contains('modal')) {
            const modalId = e.target.id;
            const modal = document.getElementById(modalId);
            if (!modal) return;
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