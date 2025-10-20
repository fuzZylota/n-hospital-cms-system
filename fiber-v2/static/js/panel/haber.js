/**
 * Haber (Individual News View) Page JavaScript
 * Handles page interactions, delete functionality, and image zoom
 */

class HaberPageManager {
    constructor() {
        this.haberData = window.haberData || {};
        this.deleteModal = null;
        this.imageModal = null;
        this.isDeleting = false;
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.initializeModals();
        this.setupImageZoom();
    }

    setupEventListeners() {
        // Delete button click
        const deleteBtn = document.getElementById('deleteHaberBtn');
        if (deleteBtn) {
            deleteBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.showDeleteModal();
            });
        }

        // Confirm delete button
        const confirmDeleteBtn = document.getElementById('confirmDeleteBtn');
        if (confirmDeleteBtn) {
            confirmDeleteBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.confirmDelete();
            });
        }

        // Image zoom functionality
        const zoomableImages = document.querySelectorAll('.zoomable');
        zoomableImages.forEach(img => {
            img.addEventListener('click', (e) => {
                e.preventDefault();
                this.showImageModal(img.src, img.alt || 'Görsel');
            });
        });
    }

    initializeModals() {
        this.deleteModal = document.getElementById('deleteModal');
        this.imageModal = document.getElementById('imageZoomModal');
    }

    setupImageZoom() {
        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            if (e.target === this.deleteModal) {
                this.closeDeleteModal();
            }
            if (e.target === this.imageModal) {
                this.closeImageModal();
            }
        });

        // Close modals with Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeDeleteModal();
                this.closeImageModal();
            }
        });
    }

    showDeleteModal() {
        if (!this.deleteModal) return;

        const deleteItemName = document.getElementById('deleteItemName');
        if (deleteItemName) {
            deleteItemName.textContent = this.haberData.title || 'Bu haber';
        }

        this.deleteModal.style.display = 'flex';
        setTimeout(() => {
            this.deleteModal.classList.add('show');
        }, 10);
    }

    closeDeleteModal() {
        if (!this.deleteModal) return;

        this.deleteModal.classList.remove('show');
        setTimeout(() => {
            this.deleteModal.style.display = 'none';
        }, 300);
    }

    async confirmDelete() {
        if (this.isDeleting) return;

        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        this.isDeleting = true;
        confirmBtn.disabled = true;
        btnText.style.display = 'none';
        btnLoader.style.display = 'flex';

        try {
            const response = await fetch(`/backend/news/${this.haberData.id}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            const result = await response.json();

            if (result.status === 201) {
                this.showAlert('successMessage', 'Haber başarıyla silindi.');
                setTimeout(() => {
                    window.location.href = '/panel/haberler';
                }, 2000);
            } else {
                this.showAlert('errorMessage', result.message || 'Bir hata oluştu. Lütfen tekrar deneyin.');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showAlert('errorMessage', 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
        } finally {
            this.isDeleting = false;
            confirmBtn.disabled = false;
            btnText.style.display = 'inline';
            btnLoader.style.display = 'none';
            this.closeDeleteModal();
        }
    }

    showImageModal(src, title) {
        if (!this.imageModal) return;

        const zoomedImage = document.getElementById('zoomedImage');
        const imageTitle = document.getElementById('imageTitle');

        if (zoomedImage) {
            zoomedImage.src = src;
            zoomedImage.alt = title;
        }

        if (imageTitle) {
            imageTitle.textContent = title;
        }

        this.imageModal.style.display = 'flex';
        setTimeout(() => {
            this.imageModal.classList.add('show');
        }, 10);
    }

    closeImageModal() {
        if (!this.imageModal) return;

        this.imageModal.classList.remove('show');
        setTimeout(() => {
            this.imageModal.style.display = 'none';
        }, 300);
    }

    showAlert(alertId, message) {
        const alert = document.getElementById(alertId);
        if (!alert) return;

        const alertText = alert.querySelector('.alert-text');
        if (alertText) {
            alertText.textContent = message;
        }

        alert.style.display = 'flex';
        setTimeout(() => {
            alert.classList.add('show');
        }, 10);

        // Auto hide after 5 seconds
        setTimeout(() => {
            this.hideAlert(alertId);
        }, 5000);
    }

    hideAlert(alertId) {
        const alert = document.getElementById(alertId);
        if (!alert) return;

        alert.classList.remove('show');
        setTimeout(() => {
            alert.style.display = 'none';
        }, 300);
    }
}

// Global functions for modal interactions
function closeDeleteModal() {
    if (window.haberPageManager) {
        window.haberPageManager.closeDeleteModal();
    }
}

function closeImageModal() {
    if (window.haberPageManager) {
        window.haberPageManager.closeImageModal();
    }
}

function closeAlert(alertId) {
    if (window.haberPageManager) {
        window.haberPageManager.hideAlert(alertId);
    }
}

function downloadImage() {
    const zoomedImage = document.getElementById('zoomedImage');
    if (!zoomedImage || !zoomedImage.src) return;

    const link = document.createElement('a');
    link.href = zoomedImage.src;
    link.download = zoomedImage.alt || 'görsel';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
}

function openImageInNewTab() {
    const zoomedImage = document.getElementById('zoomedImage');
    if (!zoomedImage || !zoomedImage.src) return;

    window.open(zoomedImage.src, '_blank');
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.haberPageManager = new HaberPageManager();
});

// Export for potential external use
if (typeof module !== 'undefined' && module.exports) {
    module.exports = HaberPageManager;
}
