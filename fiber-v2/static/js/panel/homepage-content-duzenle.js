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

