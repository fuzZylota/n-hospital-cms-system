/**
 * Job Application (Individual Job Application View) Page JavaScript
 * Handles delete functionality, modal interactions, and AJAX operations
 */

class JobApplicationViewManager {
    constructor() {
        this.jobApplicationData = window.jobApplicationData || {};
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
        this.setupDropdown();
        this.setupMarkAsReadButton();
        this.setupDocumentPreview();
        this.setupKeyboardShortcuts();
        this.currentDocumentPath = null;
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');
                const documentModal = document.getElementById('documentPreviewModal');

                if (documentModal && documentModal.style.display !== 'none') {
                    this.closeDocumentPreviewModal();
                } else if (deleteModal && deleteModal.style.display !== 'none') {
                    this.closeDeleteModal();
                } else {
                    this.closeDropdown();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const documentModal = document.getElementById('documentPreviewModal');

            if (e.target === documentModal) {
                this.closeDocumentPreviewModal();
            } else if (e.target === deleteModal) {
                this.closeDeleteModal();
            }
        });

        // Handle window resize
        window.addEventListener('resize', () => {
            this.handleResize();
        });
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteJobApplicationBtn');
        if (!deleteBtn) return;
        
        deleteBtn.addEventListener('click', (e) => {
            e.preventDefault();
            this.showDeleteModal();
        });
        
        // Setup confirm delete button
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (confirmBtn) {
            confirmBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.deleteJobApplication();
            });
        }
    }

    /**
     * Setup dropdown functionality
     */
    setupDropdown() {
        const dropdownToggle = document.getElementById('jobApplicationOptionsDropdown');
        const dropdownMenu = document.querySelector('#jobApplicationOptionsDropdown + .dropdown-menu');
        
        if (!dropdownToggle || !dropdownMenu) return;
        
        dropdownToggle.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleDropdown();
        });
        
        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!dropdownToggle.contains(e.target) && !dropdownMenu.contains(e.target)) {
                this.closeDropdown();
            }
        });
    }

    /**
     * Setup mark as read button functionality
     */
    setupMarkAsReadButton() {
        const markAsReadBtn = document.getElementById('markAsReadBtn');
        if (!markAsReadBtn) return;
        
        markAsReadBtn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            this.toggleReadStatus();
        });
    }

    /**
     * Setup document preview functionality
     */
    setupDocumentPreview() {
        const previewButtons = document.querySelectorAll('.btn-preview-cv');
        previewButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const filePath = button.getAttribute('data-filepath');
                const fileName = button.getAttribute('data-filename') || 'Doküman';
                const fileType = button.getAttribute('data-filetype') || '';
                this.openDocumentPreview(filePath, fileName, fileType);
            });
        });
    }

    /**
     * Open document preview modal
     */
    openDocumentPreview(filePath, fileName, fileType) {
        const modal = document.getElementById('documentPreviewModal');
        const container = document.getElementById('documentPreviewContainer');
        const title = document.getElementById('documentPreviewTitle');
        const info = document.getElementById('documentPreviewInfo');
        
        if (!modal || !container) return;

        // Store current document path
        this.currentDocumentPath = filePath;

        // Set title and info
        if (title) title.textContent = fileName;
        if (info) info.textContent = fileName;

        // Clear previous content
        container.innerHTML = '';

        // Determine file type from extension
        const fileExtension = fileType.replace(/^\./, '').toLowerCase();
        
        // Handle different file types
        if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(fileExtension)) {
            this.previewImage(container, filePath);
        } else if (['mp4', 'webm', 'ogg'].includes(fileExtension)) {
            this.previewVideo(container, filePath);
        } else if (['mp3', 'wav', 'ogg'].includes(fileExtension)) {
            this.previewAudio(container, filePath);
        } else if (fileExtension === 'pdf') {
            this.previewPDF(container, filePath);
        } else if (['txt', 'md', 'html', 'css', 'js'].includes(fileExtension)) {
            this.previewText(container, filePath);
        } else {
            this.previewUnsupported(container, fileExtension);
        }

        // Show modal
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Encode file path properly, handling spaces and special characters in filename
     */
    encodeFilePath(filePath) {
        // Convert relative path to absolute path
        let absolutePath = filePath;
        if (filePath.startsWith('../')) {
            // Remove relative path markers and make it absolute
            absolutePath = '/' + filePath.replace(/^\.\.\//g, '');
        } else if (!filePath.startsWith('/')) {
            // If it doesn't start with /, add it
            absolutePath = '/' + filePath;
        }
        
        // Split path into directory and filename
        const lastSlash = absolutePath.lastIndexOf('/');
        if (lastSlash === -1) {
            // No directory, just filename - encode it
            return '/' + encodeURIComponent(absolutePath);
        }
        
        const directory = absolutePath.substring(0, lastSlash + 1);
        const filename = absolutePath.substring(lastSlash + 1);
        
        // Encode only the filename part, keep directory as-is
        return directory + encodeURIComponent(filename);
    }

    /**
     * Preview image files
     */
    previewImage(container, filePath) {
        const encodedPath = this.encodeFilePath(filePath);
        const img = document.createElement('img');
        img.src = encodedPath;
        img.alt = 'Doküman Önizleme';
        img.style.maxWidth = '100%';
        img.style.maxHeight = '100%';
        img.style.objectFit = 'contain';
        container.appendChild(img);
    }

    /**
     * Preview video files
     */
    previewVideo(container, filePath) {
        const encodedPath = this.encodeFilePath(filePath);
        const video = document.createElement('video');
        video.src = encodedPath;
        video.controls = true;
        video.style.maxWidth = '100%';
        video.style.maxHeight = '100%';
        video.style.objectFit = 'contain';
        container.appendChild(video);
    }

    /**
     * Preview audio files
     */
    previewAudio(container, filePath) {
        const encodedPath = this.encodeFilePath(filePath);
        const audio = document.createElement('audio');
        audio.src = encodedPath;
        audio.controls = true;
        audio.style.width = '100%';
        audio.style.maxWidth = '500px';
        container.appendChild(audio);
    }

    /**
     * Preview PDF files
     */
    previewPDF(container, filePath) {
        const iframe = document.createElement('iframe');
        iframe.src = filePath;
        iframe.style.width = '100%';
        iframe.style.height = '100%';
        iframe.style.border = 'none';
        container.appendChild(iframe);
    }

    /**
     * Preview text files
     */
    previewText(container, filePath) {
        const encodedPath = this.encodeFilePath(filePath);
        const iframe = document.createElement('iframe');
        iframe.src = encodedPath;
        iframe.style.width = '100%';
        iframe.style.height = '100%';
        iframe.style.border = 'none';
        container.appendChild(iframe);
    }

    /**
     * Show unsupported file type message
     */
    previewUnsupported(container, fileExtension) {
        container.innerHTML = `
            <div class="unsupported-file">
                <i class="fas fa-file"></i>
                <h3>Önizleme Desteklenmiyor</h3>
                <p>Bu dosya türü (.${fileExtension}) önizlenemiyor. Dosyayı indirmek için "İndir" butonunu kullanın.</p>
            </div>
        `;
    }

    /**
     * Close document preview modal
     */
    closeDocumentPreviewModal() {
        const modal = document.getElementById('documentPreviewModal');
        if (!modal) return;

        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
        
        // Clear current document path
        this.currentDocumentPath = null;
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Download current document
     */
    downloadDocument() {
        if (this.currentDocumentPath) {
            const encodedPath = this.currentDocumentPath.replace(/\?preview=1$/, '');
            const link = document.createElement('a');
            link.href = encodedPath;
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        }
    }

    /**
     * Open document in new tab
     */
    openDocumentInNewTab() {
        if (this.currentDocumentPath) {
            window.open(this.currentDocumentPath, '_blank');
        }
    }

    /**
     * Toggle dropdown menu
     */
    toggleDropdown() {
        const dropdownMenu = document.querySelector('#jobApplicationOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('jobApplicationOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        const isOpen = dropdownMenu.classList.contains('show');
        
        if (isOpen) {
            this.closeDropdown();
        } else {
            this.openDropdown();
        }
    }

    /**
     * Open dropdown menu
     */
    openDropdown() {
        const dropdownMenu = document.querySelector('#jobApplicationOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('jobApplicationOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.add('show');
        dropdownToggle.setAttribute('aria-expanded', 'true');
    }

    /**
     * Close dropdown menu
     */
    closeDropdown() {
        const dropdownMenu = document.querySelector('#jobApplicationOptionsDropdown + .dropdown-menu');
        const dropdownToggle = document.getElementById('jobApplicationOptionsDropdown');
        
        if (!dropdownMenu || !dropdownToggle) return;
        
        dropdownMenu.classList.remove('show');
        dropdownToggle.setAttribute('aria-expanded', 'false');
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteModal() {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        
        if (!modal) return;
        
        if (nameSpan) {
            nameSpan.textContent = this.jobApplicationData.name || 'Bu iş başvurusu';
        }
        
        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close delete modal
     */
    closeDeleteModal() {
        const modal = document.getElementById('deleteModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
    }

    /**
     * Toggle read status
     */
    async toggleReadStatus() {
        const markAsReadBtn = document.getElementById('markAsReadBtn');
        if (!markAsReadBtn) return;
        
        this.setButtonLoading(markAsReadBtn, true);
        
        try {
            const response = await fetch(`/backend/job-application/${this.jobApplicationData.id}/set-as-read`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    is_read: !this.jobApplicationData.isRead
                })
            });
            
            const data = await response.json();
            
            if (data.status === 201) {
                this.showAlert(data.message || 'Durum başarıyla güncellendi.', 'success');
                this.jobApplicationData.isRead = !this.jobApplicationData.isRead;
                
                // Update button text
                const buttonText = this.jobApplicationData.isRead ? 'Okunmadı Olarak İşaretle' : 'Okundu Olarak İşaretle';
                markAsReadBtn.innerHTML = `<i class="fas fa-check"></i> ${buttonText}`;
                
                // Update status badge
                this.updateStatusBadge();
                
                // Close dropdown
                this.closeDropdown();
            } else {
                this.showAlert(data.message || 'Durum güncellenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error toggling read status:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(markAsReadBtn, false);
        }
    }

    /**
     * Update status badge
     */
    updateStatusBadge() {
        const statusBadges = document.querySelectorAll('.status-badge');
        statusBadges.forEach(badge => {
            if (badge.classList.contains('read') || badge.classList.contains('unread')) {
                if (this.jobApplicationData.isRead) {
                    badge.classList.remove('unread');
                    badge.classList.add('read');
                    badge.innerHTML = '<i class="fas fa-check-circle"></i> Okundu';
                } else {
                    badge.classList.remove('read');
                    badge.classList.add('unread');
                    badge.innerHTML = '<i class="fas fa-circle"></i> Okunmadı';
                }
            }
        });
    }

    /**
     * Delete job application
     */
    async deleteJobApplication() {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (!confirmBtn) return;
        
        this.setButtonLoading(confirmBtn, true);
        
        try {
            const response = await fetch(`/backend/job-application/${this.jobApplicationData.id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 201) {
                this.showAlert('İş başvurusu başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.href = '/panel/is-basvurulari';
                }, 1500);
            } else {
                this.showAlert(data.message || 'İş başvurusu silinirken hata oluştu.', 'error');
                this.setButtonLoading(confirmBtn, false);
            }
        } catch (error) {
            console.error('Error deleting job application:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            this.setButtonLoading(confirmBtn, false);
        }
    }

    /**
     * Set button loading state
     */
    setButtonLoading(button, loading) {
        if (!button) return;
        
        const textSpan = button.querySelector('.btn-text');
        const loader = button.querySelector('.btn-loader');
        
        if (loading) {
            button.disabled = true;
            if (textSpan) textSpan.style.display = 'none';
            if (loader) loader.style.display = 'inline-flex';
        } else {
            button.disabled = false;
            if (textSpan) textSpan.style.display = 'inline';
            if (loader) loader.style.display = 'none';
        }
    }

    /**
     * Show alert message
     */
    showAlert(message, type = 'success') {
        const alertId = type === 'success' ? 'successMessage' : 'errorMessage';
        const alert = document.getElementById(alertId);
        
        if (!alert) return;
        
        const textSpan = alert.querySelector('.alert-text');
        if (textSpan) {
            textSpan.textContent = message;
        }
        
        alert.style.display = 'flex';
        
        // Auto hide after 5 seconds
        setTimeout(() => {
            this.closeAlert(alertId);
        }, 5000);
    }

    /**
     * Close alert message
     */
    closeAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (alert) {
            alert.style.display = 'none';
        }
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                this.showDeleteModal();
            }
            
            // Ctrl/Cmd + E for edit
            if ((e.ctrlKey || e.metaKey) && e.key === 'e') {
                e.preventDefault();
                const editLink = document.querySelector('a[href*="/duzenle"]');
                if (editLink) {
                    editLink.click();
                }
            }
            
            // Ctrl/Cmd + R for mark as read
            if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
                e.preventDefault();
                this.toggleReadStatus();
            }
        });
    }

    /**
     * Handle window resize
     */
    handleResize() {
        // Adjust modal positions if needed
        const modals = document.querySelectorAll('.modal');
        modals.forEach(modal => {
            if (modal.style.display !== 'none') {
                // Recalculate modal position if needed
            }
        });
    }
}

// Global functions for HTML onclick attributes
function closeDeleteModal() {
    if (window.jobApplicationViewManager) {
        window.jobApplicationViewManager.closeDeleteModal();
    }
}

function closeDocumentPreviewModal() {
    if (window.jobApplicationViewManager) {
        window.jobApplicationViewManager.closeDocumentPreviewModal();
    }
}

function downloadDocument() {
    if (window.jobApplicationViewManager) {
        window.jobApplicationViewManager.downloadDocument();
    }
}

function openDocumentInNewTab() {
    if (window.jobApplicationViewManager) {
        window.jobApplicationViewManager.openDocumentInNewTab();
    }
}

function closeAlert(alertId) {
    if (window.jobApplicationViewManager) {
        window.jobApplicationViewManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.jobApplicationViewManager = new JobApplicationViewManager();
});

