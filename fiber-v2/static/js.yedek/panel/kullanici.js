/**
 * Kullanici (Individual User View) Page JavaScript
 * Handles delete functionality, password change, copy to clipboard, and interactive elements
 */

class UserViewManager {
    constructor() {
        this.userData = window.userData || {};
        this.passwordModal = document.getElementById('passwordChangeModal');
        this.passwordForm = document.getElementById('passwordChangeForm');
        this.passwordInput = document.getElementById('new_password');
        this.confirmPasswordInput = document.getElementById('confirm_password');
        this.passwordStrength = document.getElementById('passwordStrengthModal');
        this.passwordMatchIndicator = document.getElementById('passwordMatchIndicator');
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
        this.setupPasswordChangeButtons();
        this.setupPasswordHandling();
        this.setupCopyToClipboard();
        this.setupPasswordToggles();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Handle escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                const deleteModal = document.getElementById('deleteModal');
                const passwordModal = document.getElementById('passwordChangeModal');
                
                if (deleteModal.classList.contains('show')) {
                    this.closeDeleteModal();
                } else if (passwordModal.classList.contains('show')) {
                    this.closePasswordModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const passwordModal = document.getElementById('passwordChangeModal');
            
            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === passwordModal) {
                this.closePasswordModal();
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
        const deleteBtn = document.getElementById('deleteUserBtn') || document.getElementById('deleteIndividualUserBtn');
        
        if (!deleteBtn) return;

        deleteBtn.addEventListener('click', () => {
            const id = deleteBtn.getAttribute('data-id');
            const name = deleteBtn.getAttribute('data-name');
            this.showDeleteConfirmation(id, name);
        });
    }

    /**
     * Setup password change buttons
     */
    setupPasswordChangeButtons() {
        const changePasswordBtn = document.getElementById('changePasswordBtn');
        const changePasswordBtnInline = document.getElementById('changePasswordBtnInline');
        const confirmPasswordChangeBtn = document.getElementById('confirmPasswordChangeBtn');

        if (changePasswordBtn) {
            changePasswordBtn.addEventListener('click', () => {
                this.showPasswordChangeModal();
            });
        }

        if (changePasswordBtnInline) {
            changePasswordBtnInline.addEventListener('click', () => {
                this.showPasswordChangeModal();
            });
        }

        if (confirmPasswordChangeBtn) {
            confirmPasswordChangeBtn.addEventListener('click', () => {
                this.changePassword();
            });
        }
    }

    /**
     * Setup password handling and strength indicator
     */
    setupPasswordHandling() {
        if (!this.passwordInput || !this.confirmPasswordInput) return;

        // Show password strength when password input is focused
        this.passwordInput.addEventListener('focus', () => {
            this.passwordStrength.style.display = 'block';
        });

        // Update password strength on input
        this.passwordInput.addEventListener('input', () => {
            this.updatePasswordStrength();
            this.validatePasswordMatch();
        });

        // Validate password match on confirm input
        this.confirmPasswordInput.addEventListener('input', () => {
            this.validatePasswordMatch();
        });
    }

    /**
     * Setup password toggle buttons
     */
    setupPasswordToggles() {
        const toggleButtons = document.querySelectorAll('.password-toggle');
        
        toggleButtons.forEach(button => {
            button.addEventListener('click', () => {
                const targetId = button.getAttribute('data-target');
                const targetInput = document.getElementById(targetId);
                const icon = button.querySelector('i');
                
                if (targetInput.type === 'password') {
                    targetInput.type = 'text';
                    icon.className = 'fas fa-eye-slash';
                } else {
                    targetInput.type = 'password';
                    icon.className = 'fas fa-eye';
                }
            });
        });
    }

    /**
     * Update password strength indicator
     */
    updatePasswordStrength() {
        const password = this.passwordInput.value;
        const strength = this.calculatePasswordStrength(password);
        
        const strengthFill = this.passwordStrength.querySelector('.strength-fill');
        const strengthLevel = this.passwordStrength.querySelector('.strength-level');
        
        // Remove all strength classes
        strengthFill.classList.remove('weak', 'fair', 'good', 'strong', 'very-strong');
        strengthLevel.classList.remove('weak', 'fair', 'good', 'strong', 'very-strong');
        
        // Add current strength class
        strengthFill.classList.add(strength.level);
        strengthLevel.classList.add(strength.level);
        strengthLevel.textContent = strength.text;
        
        // Update requirements
        this.updatePasswordRequirements(password);
    }

    /**
     * Calculate password strength
     */
    calculatePasswordStrength(password) {
        let score = 0;
        
        // Length check
        if (password.length >= 6) score += 1;
        if (password.length >= 8) score += 1;
        if (password.length >= 12) score += 1;
        
        // Character variety checks
        if (/[a-z]/.test(password)) score += 1;
        if (/[A-Z]/.test(password)) score += 1;
        if (/[0-9]/.test(password)) score += 1;
        if (/[^A-Za-z0-9]/.test(password)) score += 1;
        
        // Determine strength level
        if (score < 3) {
            return { level: 'weak', text: 'Zayıf' };
        } else if (score < 5) {
            return { level: 'fair', text: 'Orta' };
        } else if (score < 6) {
            return { level: 'good', text: 'İyi' };
        } else if (score < 7) {
            return { level: 'strong', text: 'Güçlü' };
        } else {
            return { level: 'very-strong', text: 'Çok Güçlü' };
        }
    }

    /**
     * Update password requirements indicators
     */
    updatePasswordRequirements(password) {
        const requirements = {
            length: password.length >= 6,
            uppercase: /[A-Z]/.test(password),
            lowercase: /[a-z]/.test(password),
            number: /[0-9]/.test(password),
            special: /[^A-Za-z0-9]/.test(password)
        };

        Object.keys(requirements).forEach(req => {
            const element = this.passwordStrength.querySelector(`[data-requirement="${req}"]`);
            const icon = element.querySelector('i');
            
            if (requirements[req]) {
                element.classList.add('met');
                icon.className = 'fas fa-check';
            } else {
                element.classList.remove('met');
                icon.className = 'fas fa-times';
            }
        });
    }

    /**
     * Validate password match
     */
    validatePasswordMatch() {
        const password = this.passwordInput.value;
        const confirmPassword = this.confirmPasswordInput.value;
        
        if (!confirmPassword) {
            this.passwordMatchIndicator.style.display = 'none';
            return;
        }
        
        const isMatch = password === confirmPassword;
        
        this.passwordMatchIndicator.style.display = 'flex';
        this.passwordMatchIndicator.className = `password-match-indicator ${isMatch ? 'match' : 'no-match'}`;
        this.passwordMatchIndicator.innerHTML = `
            <i class="fas fa-${isMatch ? 'check' : 'times'}"></i>
            <span>${isMatch ? 'Şifreler eşleşiyor' : 'Şifreler eşleşmiyor'}</span>
        `;
        
        // Update field state
        this.setFieldState(this.confirmPasswordInput, isMatch);
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid) {
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');
    }

    /**
     * Show password change modal
     */
    showPasswordChangeModal() {
        this.passwordModal.style.display = 'flex';
        setTimeout(() => this.passwordModal.classList.add('show'), 10);
        
        // Reset form
        this.passwordForm.reset();
        this.passwordStrength.style.display = 'none';
        this.passwordMatchIndicator.style.display = 'none';
        
        // Reset field states
        this.passwordInput.classList.remove('error', 'success');
        this.confirmPasswordInput.classList.remove('error', 'success');
        
        // Reset password toggles
        const passwordInputs = this.passwordModal.querySelectorAll('input[type="text"][id*="password"]');
        passwordInputs.forEach(input => {
            input.type = 'password';
        });
        
        const toggleIcons = this.passwordModal.querySelectorAll('.password-toggle i');
        toggleIcons.forEach(icon => {
            icon.className = 'fas fa-eye';
        });
        
        // Focus first input
        setTimeout(() => this.passwordInput.focus(), 100);
        
        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Close password change modal
     */
    closePasswordModal() {
        this.passwordModal.classList.remove('show');
        setTimeout(() => {
            this.passwordModal.style.display = 'none';
        }, 300);
        
        // Restore body scroll
        document.body.style.overflow = '';
    }

    /**
     * Change password
     */
    async changePassword() {
        const password = this.passwordInput.value;
        const confirmPassword = this.confirmPasswordInput.value;
        
        // Validate inputs
        if (!password || password.length < 6) {
            this.showAlert('Şifre en az 6 karakter olmalıdır.', 'error');
            this.setFieldState(this.passwordInput, false);
            return;
        }
        
        if (password !== confirmPassword) {
            this.showAlert('Şifreler eşleşmiyor.', 'error');
            this.setFieldState(this.confirmPasswordInput, false);
            return;
        }

        const confirmBtn = document.getElementById('confirmPasswordChangeBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/user/${this.userData.id}/change-password`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    password: password,
                    password_confirm: confirmPassword
                })
            });

            const result = await response.json().catch(() => ({}));

            if (result.status === 201) {
                this.closePasswordModal();
                this.showAlert('Şifre başarıyla değiştirildi.', 'success');
            } else {
                this.showAlert(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Password change error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            confirmBtn.disabled = false;
            btnText.style.opacity = '1';
            btnLoader.style.display = 'none';
        }
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
            this.deleteUser(id);
        });

        // Prevent body scroll
        document.body.style.overflow = 'hidden';
    }

    /**
     * Delete user
     */
    async deleteUser(id) {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        const btnText = confirmBtn.querySelector('.btn-text');
        const btnLoader = confirmBtn.querySelector('.btn-loader');

        if (!confirmBtn || !btnText || !btnLoader) return;

        try {
            // Show loading state
            confirmBtn.disabled = true;
            btnText.style.opacity = '0';
            btnLoader.style.display = 'block';

            const response = await fetch(`/backend/user/${id}/delete`, {
                method: 'POST',
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const responseData = await response.json().catch(() => ({}));
            
            if (responseData.status === 201) {
                this.closeDeleteModal();
                this.showAlert('Kullanıcı başarıyla silindi.', 'success');
                
                // Redirect to users list after 2 seconds
                setTimeout(() => {
                    window.location.href = '/panel/kullanicilar';
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
        const copyButtons = document.querySelectorAll('.copy-btn');
        
        copyButtons.forEach(button => {
            button.addEventListener('click', () => {
                const textToCopy = button.getAttribute('data-copy');
                this.copyToClipboard(textToCopy, 'Kopyalandı!');
            });
        });
    }

    /**
     * Copy text to clipboard
     */
    async copyToClipboard(text, successMessage = 'Kopyalandı!') {
        try {
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
            
            this.showAlert(successMessage, 'success');
        } catch (error) {
            console.error('Copy to clipboard failed:', error);
            this.showAlert('Kopyalama işlemi başarısız oldu.', 'error');
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
        // Close any open modals on mobile
        if (window.innerWidth < 768) {
            const passwordModal = document.getElementById('passwordChangeModal');
            if (passwordModal?.classList.contains('show')) {
                this.closePasswordModal();
            }
        }
    }

    /**
     * Format bytes for display
     */
    formatBytes(bytes) {
        if (bytes === 0) return '0 Bytes';
        
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
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
            
            // Ctrl/Cmd + P for password change
            if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
                e.preventDefault();
                this.showPasswordChangeModal();
            }
            
            // Ctrl/Cmd + D for delete
            if ((e.ctrlKey || e.metaKey) && e.key === 'd') {
                e.preventDefault();
                const deleteBtn = document.getElementById('deleteUserBtn') || document.getElementById('deleteIndividualUserBtn');
                if (deleteBtn) {
                    deleteBtn.click();
                }
            }
            
            // Ctrl/Cmd + B for back to list
            if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
                e.preventDefault();
                window.location.href = '/panel/kullanicilar';
            }
        });
    }
}

/**
 * Global functions for template usage
 */
window.closeDeleteModal = function() {
    const modal = document.getElementById('deleteModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.closePasswordModal = function() {
    const modal = document.getElementById('passwordChangeModal');
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
    }, 300);
    document.body.style.overflow = '';
};

window.closeAlert = function(alertId) {
    const alert = document.getElementById(alertId);
    alert.classList.remove('show');
    setTimeout(() => {
        alert.style.display = 'none';
    }, 300);
};

/**
 * Utility functions
 */
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('tr-TR', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

function generateStrongPassword() {
    const length = 12;
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?';
    let password = '';
    
    // Ensure at least one character from each required category
    const categories = [
        'abcdefghijklmnopqrstuvwxyz',
        'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
        '0123456789',
        '!@#$%^&*()_+-='
    ];
    
    categories.forEach(category => {
        password += category.charAt(Math.floor(Math.random() * category.length));
    });
    
    // Fill remaining length with random characters
    for (let i = password.length; i < length; i++) {
        password += charset.charAt(Math.floor(Math.random() * charset.length));
    }
    
    // Shuffle the password
    return password.split('').sort(() => 0.5 - Math.random()).join('');
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.userViewManager = new UserViewManager();
    
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
window.UserViewManager = UserViewManager;
