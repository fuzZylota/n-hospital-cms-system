/**
 * Kullanici Ekle (User Add) Page JavaScript
 * Handles form submission, validation, password strength, and user interactions
 */

class UserFormHandler {
    constructor() {
        this.form = document.getElementById('userForm');
        this.submitBtn = document.getElementById('submitBtn');
        this.passwordInput = document.getElementById('password');
        this.passwordConfirmInput = document.getElementById('password_confirm');
        this.passwordStrength = document.getElementById('passwordStrength');
        
        this.init();
    }

    init() {
        this.setupFormSubmission();
        this.setupPasswordHandling();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupPasswordToggles();
        this.handleUrlErrors();
    }

    /**
     * Setup form submission
     */
    setupFormSubmission() {
        this.form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateForm()) {
                return;
            }

            await this.submitForm();
        });
    }

    /**
     * Setup password handling and strength indicator
     */
    setupPasswordHandling() {
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
        this.passwordConfirmInput.addEventListener('input', () => {
            this.validatePasswordMatch();
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
     * Highlight specific password requirement based on error
     */
    highlightPasswordRequirement(errorCode) {
        if (!this.passwordStrength) return;
        
        let requirementToHighlight = null;
        
        switch (errorCode) {
            case 'password_is_too_short':
                requirementToHighlight = 'length';
                break;
            case 'password_must_contain_at_least_one_uppercase_letter':
                requirementToHighlight = 'uppercase';
                break;
            case 'password_must_contain_at_least_one_lowercase_letter':
                requirementToHighlight = 'lowercase';
                break;
            case 'password_must_contain_at_least_one_number':
                requirementToHighlight = 'number';
                break;
            case 'password_must_contain_at_least_one_special_character':
                requirementToHighlight = 'special';
                break;
        }
        
        if (requirementToHighlight) {
            const element = this.passwordStrength.querySelector(`[data-requirement="${requirementToHighlight}"]`);
            if (element) {
                element.classList.add('error-highlight');
                setTimeout(() => {
                    element.classList.remove('error-highlight');
                }, 3000);
            }
        }
    }

    /**
     * Validate password match
     */
    validatePasswordMatch() {
        const password = this.passwordInput.value;
        const confirmPassword = this.passwordConfirmInput.value;
        
        if (!confirmPassword) return;
        
        const isMatch = password === confirmPassword;
        
        // Remove existing indicator
        const existingIndicator = this.passwordConfirmInput.parentNode.parentNode.querySelector('.password-match-indicator');
        if (existingIndicator) {
            existingIndicator.remove();
        }
        
        // Add match indicator
        const indicator = document.createElement('div');
        indicator.className = `password-match-indicator ${isMatch ? 'match' : 'no-match'}`;
        indicator.innerHTML = `
            <i class="fas fa-${isMatch ? 'check' : 'times'}"></i>
            ${isMatch ? 'Şifreler eşleşiyor' : 'Şifreler eşleşmiyor'}
        `;
        
        this.passwordConfirmInput.parentNode.parentNode.appendChild(indicator);
        
        // Update field state
        this.setFieldState(this.passwordConfirmInput, isMatch, isMatch ? '' : 'Şifreler eşleşmiyor');
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
     * Setup form validation
     */
    setupFormValidation() {
        const requiredFields = this.form.querySelectorAll('[required]');
        
        requiredFields.forEach(field => {
            field.addEventListener('blur', () => {
                this.validateField(field);
            });
            
            field.addEventListener('input', () => {
                if (field.classList.contains('error')) {
                    this.validateField(field);
                }
            });
        });

        // Email validation
        const emailField = this.form.querySelector('input[type="email"]');
        if (emailField) {
            emailField.addEventListener('blur', () => {
                this.validateEmail(emailField);
            });
        }

        // Phone validation
        const phoneField = this.form.querySelector('input[type="tel"]');
        if (phoneField) {
            phoneField.addEventListener('blur', () => {
                this.validatePhone(phoneField);
            });
        }

        // Role selection validation
        const roleField = this.form.querySelector('#role');
        if (roleField) {
            roleField.addEventListener('change', () => {
                this.validateField(roleField);
                this.updateRoleInfo(roleField.value);
            });
        }
    }

    /**
     * Setup toggle switches
     */
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
            updateToggleText(); // Initial state
        });
    }

    /**
     * Update role information
     */
    updateRoleInfo(role) {
        // This could be extended to show role-specific information
        const roleDescriptions = {
            admin: 'Tam sistem erişimi ve tüm yönetim yetkilerine sahiptir',
            moderator: 'Sınırlı yönetim yetkilerine sahiptir'
        };
        
        // Update help text if needed
        const helpText = this.form.querySelector('#role').parentNode.querySelector('.form-help');
        if (helpText && roleDescriptions[role]) {
            helpText.textContent = roleDescriptions[role];
        }
    }

    /**
     * Validate individual field
     */
    validateField(field) {
        const value = field.value.trim();
        let isValid = true;
        let errorMessage = '';

        // Required field validation
        if (field.hasAttribute('required') && !value) {
            isValid = false;
            errorMessage = 'Bu alan zorunludur.';
        }

        // Minimum length validation
        if (field.hasAttribute('minlength') && value.length > 0 && value.length < field.getAttribute('minlength')) {
            isValid = false;
            errorMessage = `En az ${field.getAttribute('minlength')} karakter olmalıdır.`;
        }

        // Maximum length validation
        if (field.hasAttribute('maxlength') && value.length > field.getAttribute('maxlength')) {
            isValid = false;
            errorMessage = `En fazla ${field.getAttribute('maxlength')} karakter olmalıdır.`;
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate email field
     */
    validateEmail(field) {
        const email = field.value.trim();
        if (!email && !field.hasAttribute('required')) return true;
        
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        const isValid = emailRegex.test(email);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir e-posta adresi girin.');
        return isValid;
    }

    /**
     * Validate phone field
     */
    validatePhone(field) {
        const phone = field.value.trim();
        if (!phone && !field.hasAttribute('required')) return true;
        
        // Basic phone validation (at least 10 digits)
        const phoneRegex = /^[\+]?[\d\s\-\(\)]{10,}$/;
        const isValid = phoneRegex.test(phone);
        
        this.setFieldState(field, isValid, isValid ? '' : 'Geçerli bir telefon numarası girin.');
        return isValid;
    }

    /**
     * Set field validation state
     */
    setFieldState(field, isValid, errorMessage) {
        const fieldGroup = field.closest('.form-group');
        let errorElement = fieldGroup.querySelector('.field-error');

        // Remove existing error
        if (errorElement) {
            errorElement.remove();
        }

        // Update field classes
        field.classList.remove('error', 'success');
        field.classList.add(isValid ? 'success' : 'error');

        // Add error message
        if (!isValid && errorMessage) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            errorElement.innerHTML = `<i class="fas fa-exclamation-circle"></i> ${errorMessage}`;
            fieldGroup.appendChild(errorElement);
        }
    }

    /**
     * Validate entire form
     */
    validateForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.form.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate email
        const emailField = this.form.querySelector('input[type="email"]');
        if (emailField && !this.validateEmail(emailField)) {
            isValid = false;
        }

        // Validate phone
        const phoneField = this.form.querySelector('input[type="tel"]');
        if (phoneField && !this.validatePhone(phoneField)) {
            isValid = false;
        }

        // Validate password match
        const password = this.passwordInput.value;
        const confirmPassword = this.passwordConfirmInput.value;
        if (password !== confirmPassword) {
            this.setFieldState(this.passwordConfirmInput, false, 'Şifreler eşleşmiyor');
            isValid = false;
        }

        // Validate password strength (minimum requirements)
        if (password.length < 6) {
            this.setFieldState(this.passwordInput, false, 'Şifre en az 6 karakter olmalıdır');
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.form.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Submit form
     */
    async submitForm() {
        try {
            this.setLoadingState(true);

            const formData = new FormData(this.form);
            
            const response = await fetch('/backend/add-user', {
                method: 'POST',
                body: formData,
                headers: {
                    'X-Requested-With': 'XMLHttpRequest'
                }
            });

            const result = await response.json().catch(() => ({}));

            if (result.status === 201) {
                this.showSuccessModal();
            } else {
                this.showErrorModal(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
            }
        } catch (error) {
            console.error('Form submission error:', error);
            this.showErrorModal('Bağlantı hatası oluştu.');
        } finally {
            this.setLoadingState(false);
        }
    }

    /**
     * Set loading state for form submission
     */
    setLoadingState(loading) {
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

    /**
     * Show success modal
     */
    showSuccessModal() {
        const modal = document.getElementById('successModal');
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Show error modal
     */
    showErrorModal(message) {
        const modal = document.getElementById('errorModal');
        const messageElement = document.getElementById('errorMessage');
        messageElement.textContent = message;
        modal.style.display = 'flex';
        setTimeout(() => modal.classList.add('show'), 10);
    }

    /**
     * Handle URL error parameters
     */
    handleUrlErrors() {
        const urlParams = new URLSearchParams(window.location.search);
        const error = urlParams.get('error');
        
        if (error) {
            const errorMessage = this.getErrorMessage(error);
            this.showErrorModal(errorMessage);
            
            // Clear URL parameters to prevent showing error on page refresh
            const newUrl = window.location.pathname;
            window.history.replaceState({}, document.title, newUrl);
            
            // Focus on relevant field if applicable
            this.focusOnErrorField(error);
        }
    }

    /**
     * Get user-friendly error message from error code
     */
    getErrorMessage(errorCode) {
        const errorMessages = {
            'only_admins_can_add_users': 'Sadece admin kullanıcılar yeni kullanıcı ekleyebilir.',
            'password_and_password_confirm_do_not_match': 'Şifre ve şifre tekrarı eşleşmiyor.',
            'internal_server_error': 'Sunucu hatası oluştu. Lütfen daha sonra tekrar deneyin.',
            'email_or_phone_already_exists': 'Bu e-posta adresi veya telefon numarası zaten kullanılıyor.',
            'password_is_too_short': 'Şifre çok kısa. En az 6 karakter olmalıdır.',
            'password_is_too_long': 'Şifre çok uzun. En fazla 255 karakter olmalıdır.',
            'password_must_contain_at_least_one_uppercase_letter': 'Şifre en az bir büyük harf içermelidir.',
            'password_must_contain_at_least_one_lowercase_letter': 'Şifre en az bir küçük harf içermelidir.',
            'password_must_contain_at_least_one_number': 'Şifre en az bir rakam içermelidir.',
            'password_must_contain_at_least_one_special_character': 'Şifre en az bir özel karakter içermelidir.'
        };
        
        return errorMessages[errorCode] || 'Bilinmeyen bir hata oluştu. Lütfen tekrar deneyin.';
    }

    /**
     * Focus on relevant field based on error type
     */
    focusOnErrorField(errorCode) {
        let fieldToFocus = null;
        let showPasswordStrength = false;
        
        switch (errorCode) {
            case 'password_and_password_confirm_do_not_match':
                fieldToFocus = this.passwordConfirmInput;
                break;
            case 'password_is_too_short':
            case 'password_is_too_long':
            case 'password_must_contain_at_least_one_uppercase_letter':
            case 'password_must_contain_at_least_one_lowercase_letter':
            case 'password_must_contain_at_least_one_number':
            case 'password_must_contain_at_least_one_special_character':
                fieldToFocus = this.passwordInput;
                showPasswordStrength = true;
                break;
            case 'email_or_phone_already_exists':
                // Try to determine if it's email or phone by checking which field has a value
                const emailField = this.form.querySelector('input[name="email"]');
                const phoneField = this.form.querySelector('input[name="phone"]');
                
                if (emailField && emailField.value.trim()) {
                    fieldToFocus = emailField;
                } else if (phoneField && phoneField.value.trim()) {
                    fieldToFocus = phoneField;
                } else {
                    fieldToFocus = emailField; // Default to email field
                }
                break;
        }
        
        if (fieldToFocus) {
            setTimeout(() => {
                // Show password strength if needed
                if (showPasswordStrength && this.passwordStrength) {
                    this.passwordStrength.style.display = 'block';
                    this.updatePasswordStrength();
                    this.highlightPasswordRequirement(errorCode);
                }
                
                // Add error class to highlight the field
                fieldToFocus.classList.add('error');
                
                // Focus and scroll to field
                fieldToFocus.focus();
                fieldToFocus.scrollIntoView({ behavior: 'smooth', block: 'center' });
                
                // Remove error class after a delay
                setTimeout(() => {
                    fieldToFocus.classList.remove('error');
                }, 3000);
            }, 500); // Delay to allow modal to show first
        }
    }

    /**
     * Reset form to initial state
     */
    resetForm() {
        this.form.reset();
        
        // Reset validation states
        const fields = this.form.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.form.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Remove password match indicators
        const matchIndicators = this.form.querySelectorAll('.password-match-indicator');
        matchIndicators.forEach(element => element.remove());
        
        // Hide password strength
        this.passwordStrength.style.display = 'none';
        
        // Reset toggle switches
        this.setupToggleSwitches();
        
        // Reset password toggles
        const passwordInputs = this.form.querySelectorAll('input[type="text"][id*="password"]');
        passwordInputs.forEach(input => {
            input.type = 'password';
        });
        
        const toggleIcons = this.form.querySelectorAll('.password-toggle i');
        toggleIcons.forEach(icon => {
            icon.className = 'fas fa-eye';
        });
    }
}

/**
 * Modal management functions
 */
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    modal.classList.remove('show');
    setTimeout(() => {
        modal.style.display = 'none';
        
        // If success modal, redirect to users list
        if (modalId === 'successModal') {
            window.location.href = '/panel/kullanicilar';
        }
    }, 300);
}

/**
 * Utility functions
 */
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

function validatePasswordStrength(password) {
    const requirements = {
        length: password.length >= 6,
        uppercase: /[A-Z]/.test(password),
        lowercase: /[a-z]/.test(password),
        number: /[0-9]/.test(password),
        special: /[^A-Za-z0-9]/.test(password)
    };
    
    const score = Object.values(requirements).filter(Boolean).length;
    
    return {
        score,
        requirements,
        strength: score < 2 ? 'weak' : score < 4 ? 'fair' : score < 5 ? 'good' : 'strong'
    };
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
    new UserFormHandler();
    
    // Close modals when clicking outside
    document.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            const modalId = e.target.id;
            closeModal(modalId);
        }
    });
    
    // Close modals with Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.show');
            if (visibleModal) {
                closeModal(visibleModal.id);
            }
        }
    });
});

// Export for potential external use
window.UserFormHandler = UserFormHandler;
