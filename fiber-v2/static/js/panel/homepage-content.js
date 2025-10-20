/**
 * Homepage Content (Individual Homepage Content View) Page JavaScript
 * Handles delete functionality, code syntax highlighting, and interactive elements
 */

class HomepageContentViewManager {
    constructor() {
        this.homepageContentData = window.homepageContentData || {};
        this.codeEditors = {};
        this.previewFrame = null;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupCodeEditors();
        this.setupDeleteButton();
        this.setupPreview();
        this.setupCopyToClipboard();
        this.setupKeyboardShortcuts();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');
                
                if (deleteModal && deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            }
        });

        // Handle window resize
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    /**
     * Setup CodeMirror editors for syntax highlighting
     */
    setupCodeEditors() {
        // HTML Editor
        const htmlTextarea = document.getElementById('htmlContent');
        if (htmlTextarea) {
            this.codeEditors.html = CodeMirror.fromTextArea(htmlTextarea, {
                mode: 'xml',
                theme: 'monokai',
                readOnly: true,
                lineNumbers: true,
                lineWrapping: true,
                foldGutter: true,
                gutters: ['CodeMirror-linenumbers', 'CodeMirror-foldgutter']
            });
        }

        // CSS Editor
        const cssTextarea = document.getElementById('cssContent');
        if (cssTextarea) {
            this.codeEditors.css = CodeMirror.fromTextArea(cssTextarea, {
                mode: 'css',
                theme: 'monokai',
                readOnly: true,
                lineNumbers: true,
                lineWrapping: true,
                foldGutter: true,
                gutters: ['CodeMirror-linenumbers', 'CodeMirror-foldgutter']
            });
        }

        // JavaScript Editor
        const jsTextarea = document.getElementById('jsContent');
        if (jsTextarea) {
            this.codeEditors.js = CodeMirror.fromTextArea(jsTextarea, {
                mode: 'javascript',
                theme: 'monokai',
                readOnly: true,
                lineNumbers: true,
                lineWrapping: true,
                foldGutter: true,
                gutters: ['CodeMirror-linenumbers', 'CodeMirror-foldgutter']
            });
        }
    }

    /**
     * Setup preview functionality
     */
    setupPreview() {
        this.previewFrame = document.getElementById('previewFrame');
        if (this.previewFrame) {
            this.refreshPreview();
        }
    }

    /**
     * Refresh the preview iframe
     */
    refreshPreview() {
        if (!this.previewFrame) return;

        const htmlContent = this.homepageContentData.htmlContent || '<p>HTML içeriği bulunmuyor</p>';
        const cssContent = this.homepageContentData.cssContent || '';
        const jsContent = this.homepageContentData.jsContent || '';

        const fullHtml = `
            <!DOCTYPE html>
            <html>
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Preview</title>
                <style>
                    body { 
                        font-family: Arial, sans-serif; 
                        margin: 20px; 
                        line-height: 1.6; 
                        color: #333;
                    }
                    ${cssContent}
                </style>
            </head>
            <body>
                ${htmlContent}
                <script>${jsContent}</script>
            </body>
            </html>
        `;

        const blob = new Blob([fullHtml], { type: 'text/html' });
        const url = URL.createObjectURL(blob);
        this.previewFrame.src = url;

        // Clean up the URL after a delay
        setTimeout(() => {
            URL.revokeObjectURL(url);
        }, 1000);
    }

    /**
     * Open preview in new tab
     */
    openPreviewInNewTab() {
        const htmlContent = this.homepageContentData.htmlContent || '<p>HTML içeriği bulunmuyor</p>';
        const cssContent = this.homepageContentData.cssContent || '';
        const jsContent = this.homepageContentData.jsContent || '';

        const fullHtml = `
            <!DOCTYPE html>
            <html>
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>Preview - ${this.homepageContentData.name}</title>
                <style>
                    body { 
                        font-family: Arial, sans-serif; 
                        margin: 20px; 
                        line-height: 1.6; 
                        color: #333;
                    }
                    ${cssContent}
                </style>
            </head>
            <body>
                ${htmlContent}
                <script>${jsContent}</script>
            </body>
            </html>
        `;

        const newWindow = window.open('', '_blank');
        newWindow.document.write(fullHtml);
        newWindow.document.close();
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteHomepageContentBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
        });
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteConfirmation(id, name) {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        const confirmBtn = document.getElementById('confirmDeleteBtn');

        if (!modal || !nameSpan || !confirmBtn) return;

        nameSpan.textContent = name;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);

        // Remove existing listeners
        const newConfirmBtn = confirmBtn.cloneNode(true);
        confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);

        // Add new listener
        newConfirmBtn.addEventListener('click', () => {
            this.deleteHomepageContent(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete homepage content
     */
    async deleteHomepageContent(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/homepage-content/${id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Anasayfa içeriği başarıyla silindi.', 'success');
                
                // Redirect to homepage contents list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/anasayfa-icerikleri';
                }, 2000);

            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
    }

    /**
     * Close delete modal
     */
    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Setup copy to clipboard functionality
     */
    setupCopyToClipboard() {
        // Add copy buttons to code sections
        const copyButtons = document.querySelectorAll('.copy-btn');
        copyButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const targetId = button.getAttribute('onclick').match(/copyToClipboard\('(\w+)'\)/)[1];
                this.copyToClipboard(targetId);
            });
        });
    }

    /**
     * Copy text to clipboard
     */
    async copyToClipboard(elementId) {
        try {
            let text = '';
            
            if (this.codeEditors[elementId.replace('Content', '')]) {
                // Get text from CodeMirror editor
                text = this.codeEditors[elementId.replace('Content', '')].getValue();
            } else {
                // Fallback to textarea
                const element = document.getElementById(elementId);
                if (element) {
                    text = element.value;
                }
            }

            if (navigator.clipboard && window.isSecureContext) {
                await navigator.clipboard.writeText(text);
            } else {
                // Fallback for older browsers
                const textArea = document.createElement('textarea');
                textArea.value = text;
                textArea.style.position = 'fixed';
                textArea.style.left = '-999999px';
                textArea.style.top = '-999999px';
                document.body.appendChild(textArea);
                textArea.focus();
                textArea.select();
                document.execCommand('copy');
                textArea.remove();
            }
            
            this.showAlert('Kod kopyalandı!', 'success');
        } catch (error) {
            console.error('Copy to clipboard failed:', error);
            this.showAlert('Kopyalama işlemi başarısız oldu.', 'error');
        }
    }

    /**
     * Toggle code expansion
     */
    toggleCodeExpansion(elementId) {
        const container = document.getElementById(elementId).closest('.code-container');
        if (container) {
            container.classList.toggle('expanded');
            const button = container.closest('.code-section').querySelector('.expand-btn i');
            if (button) {
                if (container.classList.contains('expanded')) {
                    button.className = 'fas fa-compress-arrows-alt';
                } else {
                    button.className = 'fas fa-expand-arrows-alt';
                }
            }
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        const textElement = alert?.querySelector('.alert-text');

        if (!alert || !textElement) return;

        textElement.textContent = message;
        alert.style.display = 'flex';
        setTimeout(() => alert.classList.add('show'), 10);

        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Close alert
     */
    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (!alert) return;

        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }

    /**
     * Handle window resize
     */
    handleResize() {
        // Refresh CodeMirror editors on resize
        Object.values(this.codeEditors).forEach(editor => {
            if (editor) {
                editor.refresh();
            }
        });
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + E for edit
            if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
                e.preventDefault();
                const editBtn = document.querySelector('a[href*="/duzenle"]');
                if (editBtn) {
                    editBtn.click();
                }
            }
            
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                const deleteBtn = document.getElementById('deleteHomepageContentBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/anasayfa-icerikleri';
            }
        });
    }
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    if (modal) {
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        document.body.style.overflow = '';
    }
};

window.copyToClipboard = function(elementId) {
    const manager = window.homepageContentViewManager;
    if (manager) {
        manager.copyToClipboard(elementId);
    }
};

window.toggleCodeExpansion = function(elementId) {
    const manager = window.homepageContentViewManager;
    if (manager) {
        manager.toggleCodeExpansion(elementId);
    }
};

window.refreshPreview = function() {
    const manager = window.homepageContentViewManager;
    if (manager) {
        manager.refreshPreview();
    }
};

window.openPreviewInNewTab = function() {
    const manager = window.homepageContentViewManager;
    if (manager) {
        manager.openPreviewInNewTab();
    }
};

window.closeAlert = function(alertId) {
    const alert = document.getElementById(alertId);
    if (alert) {
        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }
};

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.homepageContentViewManager = new HomepageContentViewManager();
    
    // Add CSS for animations
    const style = document.createElement('style');
    style.textContent = `
        .content-card {
            opacity: 1;
            transform: translateY(0);
            transition: all 0.6s ease;
        }
        
        .content-card.animate-in {
            opacity: 1;
            transform: translateY(0);
        }
        
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(255, 255, 255, 0.8);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 9999;
        }
        
        .loading-spinner {
            width: 40px;
            height: 40px;
            border: 4px solid var(--border-color);
            border-top: 4px solid var(--primary-color);
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
    `;
    document.head.appendChild(style);
});

// Export for potential external use
window.HomepageContentViewManager = HomepageContentViewManager;

