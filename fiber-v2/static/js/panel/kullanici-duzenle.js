/**
 * Kullanici Duzenle (User Edit) Page JavaScript
 * Handles user information form submission and validation
 */

class UserEditHandler {
    constructor() {
        this.uid = window.pageData.uid;
        this.isAdmin = window.pageData.isAdmin === true;
        this.userForm = document.getElementById('userForm');
        this.userSubmitBtn = document.getElementById('userSubmitBtn');
        
        this.init();
    }

    init() {
        this.setupUserForm();
        this.setupFormValidation();
        this.setupToggleSwitches();
        this.setupRoleHandling();
        this.loadSubeler();
    }

    /**
     * Setup user form (AJAX JSON submission)
     */
    setupUserForm() {
        this.userForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            if (!this.validateUserForm()) {
                return;
            }

            this.setUserLoadingState(true);

            try {
                // Collect all form data - simple approach
                const data = {};
                const allInputs = this.userForm.querySelectorAll('input, textarea, select');
                const selfEditFields = new Set(['uid', 'name', 'surname', 'email', 'phone', 'timezone']);
                
                allInputs.forEach(input => {
                    if (!this.isAdmin && !selfEditFields.has(input.name)) {
                        return;
                    }
                    if (input.type === 'checkbox') {
                        data[input.name] = input.checked;
                    } else if (input.type === 'hidden') {
                        // Handle hidden inputs with proper type conversion
                        if (input.value === 'true') {
                            data[input.name] = true;
                        } else if (input.value === 'false') {
                            data[input.name] = false;
                        } else {
                            data[input.name] = input.value;
                        }
                    } else {
                        data[input.name] = input.value;
                    }
                });

                const response = await fetch(`/backend/user/${this.uid}/edit`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Requested-With': 'XMLHttpRequest'
                    },
                    body: JSON.stringify(data)
                });

                const result = await response.json().catch(() => ({}));

                if (result.status === 201) {
                    this.showSuccessModal('Kullanıcı bilgileri başarıyla güncellendi.');
                    // Optionally redirect or update UI
                    setTimeout(() => {
                        window.location.href = `/panel/kullanicilar/${this.uid}`;
                    }, 2000);
                } else {
                    this.showErrorModal(result.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.');
                }

            } catch (error) {
                console.error('User form submission error:', error);
                this.showErrorModal('Bağlantı hatası oluştu.');
            } finally {
                this.setUserLoadingState(false);
            }
        });
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // Required fields validation
        const requiredFields = this.userForm.querySelectorAll('[required]');
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
        const emailField = this.userForm.querySelector('input[type="email"]');
        if (emailField) {
            emailField.addEventListener('blur', () => {
                this.validateEmail(emailField);
            });
            
            emailField.addEventListener('input', () => {
                if (emailField.classList.contains('error')) {
                    this.validateEmail(emailField);
                }
            });
        }

        // Phone validation
        const phoneField = this.userForm.querySelector('input[type="tel"]');
        if (phoneField) {
            phoneField.addEventListener('blur', () => {
                this.validatePhone(phoneField);
            });
            
            phoneField.addEventListener('input', () => {
                if (phoneField.classList.contains('error')) {
                    this.validatePhone(phoneField);
                }
            });
        }

        // Name fields validation
        const nameFields = this.userForm.querySelectorAll('input[name="name"], input[name="surname"]');
        nameFields.forEach(field => {
            field.addEventListener('input', () => {
                this.validateName(field);
            });
            
            field.addEventListener('blur', () => {
                this.validateName(field);
            });
        });
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
     * Setup role handling and information
     */
    setupRoleHandling() {
        const roleField = this.userForm.querySelector('#role');
        if (roleField) {
            roleField.addEventListener('change', () => {
                this.updateRoleInfo(roleField.value);
                this.validateField(roleField);
            });
            
            // Initialize role info
            this.updateRoleInfo(roleField.value);
        }
    }

    /**
     * Update role information
     */
    updateRoleInfo(role) {
        const roleDescriptions = {
            admin: 'Tam sistem erişimi ve tüm yönetim yetkilerine sahiptir',
            moderator: 'Sınırlı yönetim yetkilerine sahiptir',
            santral: 'Santral operatörü yetkilerine sahiptir',
            ik: 'İnsan kaynakları yetkilerine sahiptir'
        };
        
        // Update help text
        const helpText = this.userForm.querySelector('#role').parentNode.querySelector('.form-help');
        if (helpText && roleDescriptions[role]) {
            helpText.textContent = roleDescriptions[role];
        } else if (helpText) {
            helpText.textContent = 'Kullanıcının sistem yetkilerini belirler';
        }
    }

    /**
     * Load subeler from backend
     */
    async loadSubeler() {
        const select = document.getElementById('sid');
        if (!select) return;

        try {
            const response = await fetch('/backend/get-all-subeler', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200 && data.data) {
                const currentSid = window.pageData?.sid || '';
                select.innerHTML = '<option value="">Şube seçiniz...</option>';
                
                data.data.forEach(sube => {
                    const option = document.createElement('option');
                    option.value = sube.sid;
                    option.textContent = sube.name;
                    if (sube.sid === currentSid) {
                        option.selected = true;
                    }
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading subeler:', error);
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
     * Validate name fields
     */
    validateName(field) {
        const value = field.value.trim();
        const characterCount = Array.from(value).length;
        let isValid = true;
        let errorMessage = '';

        // Required validation
        if (!value) {
            isValid = false;
            errorMessage = 'Bu alan zorunludur.';
        } else if (characterCount < 2) {
            isValid = false;
            errorMessage = 'En az 2 karakter olmalıdır.';
        } else if (characterCount > 255) {
            isValid = false;
            errorMessage = 'En fazla 255 karakter olmalıdır.';
        } else if (!validatePersonName(value)) {
            isValid = false;
            errorMessage = 'Yalnızca harf, boşluk, apostrof ve tire kullanılabilir.';
        }

        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate email field
     */
    validateEmail(field) {
        const email = field.value.trim();
        let isValid = true;
        let errorMessage = '';
        
        if (!email) {
            isValid = false;
            errorMessage = 'E-posta adresi zorunludur.';
        } else if (email.length > 255) {
            isValid = false;
            errorMessage = 'E-posta adresi çok uzun.';
        } else {
            const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailRegex.test(email)) {
                isValid = false;
                errorMessage = 'Geçerli bir e-posta adresi girin.';
            }
        }
        
        this.setFieldState(field, isValid, errorMessage);
        return isValid;
    }

    /**
     * Validate phone field
     */
    validatePhone(field) {
        const phone = field.value.trim();
        let isValid = true;
        let errorMessage = '';
        
        if (!phone) {
            isValid = false;
            errorMessage = 'Telefon numarası zorunludur.';
        } else if (Array.from(phone).length > 255) {
            isValid = false;
            errorMessage = 'Telefon numarası çok uzun.';
        } else if (!validateInternationalPhone(phone)) {
            isValid = false;
            errorMessage = 'Geçerli bir telefon numarası girin.';
        }
        
        this.setFieldState(field, isValid, errorMessage);
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
     * Validate entire user form
     */
    validateUserForm() {
        let isValid = true;
        
        // Validate required fields
        const requiredFields = this.userForm.querySelectorAll('[required]');
        requiredFields.forEach(field => {
            if (!this.validateField(field)) {
                isValid = false;
            }
        });

        // Validate name fields
        const nameFields = this.userForm.querySelectorAll('input[name="name"], input[name="surname"]');
        nameFields.forEach(field => {
            if (!this.validateName(field)) {
                isValid = false;
            }
        });

        // Validate email
        const emailField = this.userForm.querySelector('input[type="email"]');
        if (emailField && !this.validateEmail(emailField)) {
            isValid = false;
        }

        // Validate phone
        const phoneField = this.userForm.querySelector('input[type="tel"]');
        if (phoneField && !this.validatePhone(phoneField)) {
            isValid = false;
        }

        if (!isValid) {
            this.showErrorModal('Lütfen tüm alanları doğru şekilde doldurun.');
            
            // Scroll to first error
            const firstError = this.userForm.querySelector('.error');
            if (firstError) {
                firstError.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstError.focus();
            }
        }

        return isValid;
    }

    /**
     * Set loading state for user form
     */
    setUserLoadingState(loading) {
        if (loading) {
            this.userSubmitBtn.disabled = true;
            this.userSubmitBtn.querySelector('.btn-text').style.opacity = '0';
            this.userSubmitBtn.querySelector('.btn-loader').style.display = 'block';
            this.userForm.classList.add('form-loading');
        } else {
            this.userSubmitBtn.disabled = false;
            this.userSubmitBtn.querySelector('.btn-text').style.opacity = '1';
            this.userSubmitBtn.querySelector('.btn-loader').style.display = 'none';
            this.userForm.classList.remove('form-loading');
        }
    }

    /**
     * Show success modal
     */
    showSuccessModal(message) {
        const modal = document.getElementById('successModal');
        const messageElement = document.getElementById('successMessage');
        messageElement.textContent = message;
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
     * Close modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        modal.classList.remove('show');
        setTimeout(() => {
            modal.style.display = 'none';
        }, 300);
    }

    /**
     * Reset form to initial state
     */
    resetForm() {
        this.userForm.reset();
        
        // Reset validation states
        const fields = this.userForm.querySelectorAll('.form-input, .form-textarea, .form-select');
        fields.forEach(field => {
            field.classList.remove('error', 'success');
        });
        
        // Remove error messages
        const errorElements = this.userForm.querySelectorAll('.field-error');
        errorElements.forEach(element => element.remove());
        
        // Reset toggle switches
        this.setupToggleSwitches();
        
        // Reset role info
        const roleField = this.userForm.querySelector('#role');
        if (roleField) {
            this.updateRoleInfo(roleField.value);
        }
    }

    /**
     * Get form data as object
     */
    getFormData() {
        const data = {};
        const inputs = this.userForm.querySelectorAll('input, textarea, select');
        
        inputs.forEach(input => {
            if (input.type === 'checkbox') {
                // Checkbox: 1 veya 0 olarak gönder, hidden input'u ezdirme
                data[input.name] = input.checked ? "1" : "0";
            } else if (input.type === 'hidden') {
                // Hidden input: sadece aynı isimde checkbox yoksa ekle
                const checkbox = this.userForm.querySelector(`input[type="checkbox"][name="${input.name}"]`);
                if (!checkbox) {
                    data[input.name] = input.value;
                }
            } else if (input.type !== 'submit' && input.type !== 'button') {
                data[input.name] = input.value;
            }
        });
        
        return data;
    }

    /**
     * Set form data from object
     */
    setFormData(data) {
        Object.keys(data).forEach(key => {
            const input = this.userForm.querySelector(`[name="${key}"]`);
            if (input) {
                if (input.type === 'checkbox') {
                    input.checked = Boolean(data[key]);
                } else {
                    input.value = data[key] || '';
                }
            }
        });
        
        // Update toggle switches
        this.setupToggleSwitches();
        
        // Update role info
        const roleField = this.userForm.querySelector('#role');
        if (roleField) {
            this.updateRoleInfo(roleField.value);
        }
    }

    /**
     * Validate unique fields (email, phone)
     */
    async validateUniqueField(fieldName, value, currentValue) {
        // Skip validation if value hasn't changed
        if (value === currentValue) {
            return true;
        }
        
        try {
            const response = await fetch(`/backend/user/validate-unique`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    field: fieldName,
                    value: value,
                    exclude_uid: this.uid
                })
            });
            
            const result = await response.json();
            return result.available === true;
        } catch (error) {
            console.error('Unique validation error:', error);
            return true; // Assume valid if validation fails
        }
    }

    /**
     * Setup real-time unique validation
     */
    setupUniqueValidation() {
        const emailField = this.userForm.querySelector('input[name="email"]');
        const phoneField = this.userForm.querySelector('input[name="phone"]');
        const oldEmail = this.userForm.querySelector('input[name="old_email"]')?.value;
        const oldPhone = this.userForm.querySelector('input[name="old_phone"]')?.value;
        
        if (emailField) {
            let emailTimeout;
            emailField.addEventListener('input', () => {
                clearTimeout(emailTimeout);
                emailTimeout = setTimeout(async () => {
                    const isUnique = await this.validateUniqueField('email', emailField.value, oldEmail);
                    if (!isUnique) {
                        this.setFieldState(emailField, false, 'Bu e-posta adresi zaten kullanılıyor.');
                    }
                }, 1000);
            });
        }
        
        if (phoneField) {
            let phoneTimeout;
            phoneField.addEventListener('input', () => {
                clearTimeout(phoneTimeout);
                phoneTimeout = setTimeout(async () => {
                    const isUnique = await this.validateUniqueField('phone', phoneField.value, oldPhone);
                    if (!isUnique) {
                        this.setFieldState(phoneField, false, 'Bu telefon numarası zaten kullanılıyor.');
                    }
                }, 1000);
            });
        }
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
    }, 300);
}

/**
 * Utility functions
 */
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

let unicodeLetterPattern = null;
try {
    unicodeLetterPattern = new RegExp('^\\p{L}$', 'u');
} catch (error) {
    // Older browsers fall back to case conversion; the server remains authoritative.
}

function isUnicodeLetter(character) {
    if (unicodeLetterPattern) {
        return unicodeLetterPattern.test(character);
    }
    return character.toLocaleUpperCase() !== character.toLocaleLowerCase();
}

function validatePersonName(name) {
    const characters = Array.from(name.trim());
    if (characters.length < 2 || characters.length > 255) {
        return false;
    }

    const separators = new Set([' ', "'", '’', '-']);
    let letterCount = 0;
    let previousWasSeparator = false;

    for (let index = 0; index < characters.length; index += 1) {
        const character = characters[index];
        if (isUnicodeLetter(character)) {
            letterCount += 1;
            previousWasSeparator = false;
            continue;
        }

        if (!separators.has(character)
            || index === 0
            || index === characters.length - 1
            || previousWasSeparator) {
            return false;
        }
        previousWasSeparator = true;
    }

    return letterCount >= 2;
}

function validateInternationalPhone(phone) {
    const characters = Array.from(phone.trim());
    let digitCount = 0;

    for (let index = 0; index < characters.length; index += 1) {
        const character = characters[index];
        if (character >= '0' && character <= '9') {
            digitCount += 1;
            continue;
        }
        if (character === ' ' || character === '(' || character === ')' || character === '-') {
            continue;
        }
        if (character === '+' && index === 0) {
            continue;
        }
        return false;
    }

    return digitCount >= 10 && digitCount <= 15;
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    const userEditHandler = new UserEditHandler();
    window.userEditHandler = userEditHandler;
    
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

    // Add keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        // Ctrl/Cmd + S for save
        if ((e.ctrlKey || e.metaKey) && e.key === 's') {
            e.preventDefault();
            const submitBtn = document.getElementById('userSubmitBtn');
            if (submitBtn && !submitBtn.disabled) {
                submitBtn.click();
            }
        }
        
        // Ctrl/Cmd + R for reset
        if ((e.ctrlKey || e.metaKey) && e.key === 'r') {
            e.preventDefault();
            if (confirm('Formu sıfırlamak istediğinizden emin misiniz?')) {
                userEditHandler.resetForm();
            }
        }
    });
});

// Export for potential external use
window.UserEditHandler = UserEditHandler;

// Sil tiklenince görüntüle otomatik seçilsin
document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('input[name^="perm_delete_"]').forEach(function(delCb) {
        const sid = delCb.name.replace('perm_delete_', '');
        const viewCb = document.querySelector('input[name="perm_view_' + sid + '"][type="checkbox"]');
        delCb.addEventListener('change', function() {
            if (delCb.checked && viewCb) viewCb.checked = true;
        });
    });
});
