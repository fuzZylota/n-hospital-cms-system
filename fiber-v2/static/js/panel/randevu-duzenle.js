/**
 * Randevu Edit Page JavaScript
 * Handles form validation, submission, and user interactions
 */

class RandevuEditManager {
    constructor() {
        this.form = document.getElementById('randevuEditForm');
        this.saveBtn = document.getElementById('saveRandevuBtn');
        this.randevuData = window.randevuData || {};
        this.originalData = {};
        this.isSubmitting = false;
        
        console.log('RandevuEditManager initialized');
        console.log('Form found:', this.form);
        console.log('Save button:', this.saveBtn);
        console.log('Randevu data:', this.randevuData);
        
        this.init();
    }

    /**
     * Initialize the manager
     */
    init() {
        this.setupEventListeners();
        this.setupFormValidation();
        this.storeOriginalData();
        this.setupDropdowns();
        this.setupTimeInput();
        this.setupAutoSave();
        this.setupKeyboardShortcuts();
    }

    /**
     * Setup all event listeners
     */
    setupEventListeners() {
        // Save button
        if (this.saveBtn) {
            console.log('Setting up save button event listener');
            this.saveBtn.addEventListener('click', (e) => {
                console.log('Save button clicked!');
                this.handleSave(e);
            });
        } else {
            console.error('Save button not found!');
        }

        // Real-time validation
        if (this.form) {
            const inputs = this.form.querySelectorAll('input, select, textarea');
            inputs.forEach(input => {
                input.addEventListener('blur', () => this.validateField(input));
                input.addEventListener('input', () => this.clearFieldError(input));
            });

            // Form change detection
            this.form.addEventListener('input', () => this.detectChanges());
            this.form.addEventListener('change', () => this.detectChanges());
        }

        // Window beforeunload
        window.addEventListener('beforeunload', (e) => this.handleBeforeUnload(e));

        // Escape key for modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeModals();
            }
        });
    }

    /**
     * Setup form validation
     */
    setupFormValidation() {
        // Add validation rules
        this.validationRules = {
            patient_first_name: {
                required: true,
                minLength: 2,
                maxLength: 100,
                pattern: /^[a-zA-ZğüşıöçĞÜŞİÖÇ\s]+$/,
                message: 'Ad en az 2, en fazla 100 karakter olmalı ve sadece harf içermelidir.'
            },
            patient_last_name: {
                required: true,
                minLength: 2,
                maxLength: 100,
                pattern: /^[a-zA-ZğüşıöçĞÜŞİÖÇ\s]+$/,
                message: 'Soyad en az 2, en fazla 100 karakter olmalı ve sadece harf içermelidir.'
            },
            patient_phone: {
                required: true,
                pattern: /^[0-9+\-\s()]+$/,
                minLength: 10,
                maxLength: 20,
                message: 'Geçerli bir telefon numarası giriniz.'
            },
            patient_email: {
                required: false,
                pattern: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
                message: 'Geçerli bir e-posta adresi giriniz.'
            },
            patient_tc_kimlik: {
                required: false,
                pattern: /^[0-9]{11}$/,
                message: 'TC Kimlik numarası 11 haneli olmalıdır.'
            },
            appointment_date: {
                required: true,
                message: 'Randevu tarihi seçiniz.'
            },
            appointment_time: {
                required: true,
                message: 'Randevu saati seçiniz.'
            },
            duration: {
                required: true,
                min: 15,
                max: 480,
                message: 'Süre 15-480 dakika arasında olmalıdır.'
            },
            price: {
                required: true,
                min: 0,
                message: 'Ücret 0 veya pozitif bir değer olmalıdır.'
            }
        };
    }

    /**
     * Setup dropdowns
     */
    setupDropdowns() {
        this.loadSubeler();
        this.setupSubeChange();
        this.setupDoktorChange();
        this.loadTibbiBirimler();
        this.loadTedkikler();
        this.loadAnlasmaliKurumlar();
    }

    /**
     * Load subeler
     */
    async loadSubeler() {
        try {
            const response = await fetch('/backend/get-all-subeler', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            console.log('Subeler response:', data);
            
            if (data.status === 200) {
                const sidSelect = document.getElementById('sid');
                if (sidSelect) {
                    sidSelect.innerHTML = '<option value="">Şube seçiniz...</option>';
                    data.data.forEach(sube => {
                        const option = document.createElement('option');
                        option.value = sube.sid;
                        option.textContent = sube.city ? `${sube.name} - ${sube.city}` : sube.name;
                        if (sube.sid === this.randevuData.sid) option.selected = true;
                        sidSelect.appendChild(option);
                    });

                    // If existing sid, cascade load and enable dependent selects
                    const bridSelect = document.getElementById('brid');
                    const dridSelect = document.getElementById('drid');
                    if (this.randevuData.sid) {
                        if (bridSelect) bridSelect.disabled = false;
                        if (dridSelect) dridSelect.disabled = false;
                        await this.loadBranches(this.randevuData.sid);
                        await this.loadDoktorlar(this.randevuData.sid);
                    } else {
                        if (bridSelect) {
                            bridSelect.disabled = true;
                            bridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                        }
                        if (dridSelect) {
                            dridSelect.disabled = true;
                            dridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                        }
                    }
                }
            }
        } catch (error) {
            console.error('Error loading subeler:', error);
        }
    }

    /**
     * Setup sube change handler
     */
    setupSubeChange() {
        const sidSelect = document.getElementById('sid');
        const bridSelect = document.getElementById('brid');
        const dridSelect = document.getElementById('drid');
        if (sidSelect) {
            sidSelect.addEventListener('change', async () => {
                const sid = sidSelect.value;
                if (sid) {
                    if (bridSelect) bridSelect.disabled = false;
                    if (dridSelect) dridSelect.disabled = false;
                    await this.loadBranches(sid);
                    await this.loadDoktorlar(sid);
                } else {
                    if (bridSelect) {
                        bridSelect.disabled = true;
                        bridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                    }
                    if (dridSelect) {
                        dridSelect.disabled = true;
                        dridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                    }
                }
            });
        }
    }

    /**
     * Load doktorlar for selected sube
     */
    async loadDoktorlar(sid) {
        try {
            const response = await fetch(`/backend/sube/${sid}/get-doctors`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            
            if (data.status === 200) {
                const dridSelect = document.getElementById('drid');
                if (dridSelect) {
                    dridSelect.innerHTML = '<option value="">Doktor Seçiniz</option>';
                    data.data.forEach(doktor => {
                        const option = document.createElement('option');
                        option.value = doktor.drid;
                        option.textContent = `${doktor.title} ${doktor.first_name} ${doktor.last_name}`;
                        if (doktor.drid === this.randevuData.drid) {
                            option.selected = true;
                        }
                        dridSelect.appendChild(option);
                    });
                    if (this.randevuData.drid) {
                        dridSelect.value = this.randevuData.drid;
                    }
                }
            }
        } catch (error) {
            console.error('Error loading doktorlar:', error);
        }
    }

    /**
     * Clear doktorlar
     */
    clearDoktorlar() {
        const dridSelect = document.getElementById('drid');
        if (dridSelect) {
            dridSelect.innerHTML = '<option value="">Doktor Seçiniz</option>';
        }
    }

    /**
     * Setup doktor change handler
     */
    setupDoktorChange() {
        const dridSelect = document.getElementById('drid');
        if (dridSelect) {
            dridSelect.addEventListener('change', () => {
                const drid = dridSelect.value;
                if (drid) {
                    this.loadBranchesForDoktor(drid);
                }
            });
        }
    }

    /**
     * Load branches for selected sube
     */
    async loadBranches(sid) {
        try {
            const response = await fetch(`/backend/sube/${sid}/get-branches`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            
            if (data.status === 200) {
                const bridSelect = document.getElementById('brid');
                if (bridSelect) {
                    bridSelect.innerHTML = '<option value="">Branş seçiniz...</option>';
                    data.data.forEach(branch => {
                        const option = document.createElement('option');
                        option.value = branch.brid;
                        option.textContent = branch.name;
                        if (branch.brid === this.randevuData.brid) {
                            option.selected = true;
                        }
                        bridSelect.appendChild(option);
                    });
                    if (this.randevuData.brid) {
                        bridSelect.value = this.randevuData.brid;
                    }
                }
            }
        } catch (error) {
            console.error('Error loading branches:', error);
        }
    }

    /**
     * Load branches for doktor
     */
    async loadBranchesForDoktor(drid) {
        // This would load branches specific to the doctor
        // Implementation depends on backend endpoint
    }

    /**
     * Clear branches
     */
    clearBranches() {
        const bridSelect = document.getElementById('brid');
        if (bridSelect) {
            bridSelect.innerHTML = '<option value="">Branş Seçiniz</option>';
        }
    }

    /**
     * Load tibbi birimler
     */
    async loadTibbiBirimler() {
        try {
            const response = await fetch('/backend/get-all-tibbi-birimler', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            
            if (data.status === 200) {
                const tbidSelect = document.getElementById('tbid');
                if (tbidSelect) {
                    tbidSelect.innerHTML = '<option value="">Tıbbi Birim Seçiniz</option>';
                    data.data.forEach(birim => {
                        const option = document.createElement('option');
                        option.value = birim.tbid;
                        option.textContent = birim.name;
                        if (birim.tbid === this.randevuData.tbid) {
                            option.selected = true;
                        }
                        tbidSelect.appendChild(option);
                    });
                }
            }
        } catch (error) {
            console.error('Error loading tibbi birimler:', error);
        }
    }

    /**
     * Load tedkikler
     */
    async loadTedkikler() {
        try {
            const response = await fetch('/backend/get-all-tedkikler', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            
            if (data.status === 200) {
                const tidSelect = document.getElementById('tid');
                if (tidSelect) {
                    tidSelect.innerHTML = '<option value="">Tetkik seçiniz...</option>';
                    data.data.forEach(tedkik => {
                        const option = document.createElement('option');
                        option.value = tedkik.tid;
                        option.textContent = tedkik.name;
                        if (tedkik.tid === this.randevuData.tid) {
                            option.selected = true;
                        }
                        tidSelect.appendChild(option);
                    });
                }
            }
        } catch (error) {
            console.error('Error loading tedkikler:', error);
        }
    }

    /**
     * Load anlasmali kurumlar
     */
    async loadAnlasmaliKurumlar() {
        try {
            const response = await fetch('/backend/get-all-anlasmali-kurumlar', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });
            const data = await response.json();
            
            if (data.status === 200) {
                const akidSelect = document.getElementById('akid');
                if (akidSelect) {
                    akidSelect.innerHTML = '<option value="">Anlaşmalı Kurum Seçiniz</option>';
                    data.data.forEach(kurum => {
                        const option = document.createElement('option');
                        option.value = kurum.akid;
                        option.textContent = kurum.name;
                        if (kurum.akid === this.randevuData.akid) {
                            option.selected = true;
                        }
                        akidSelect.appendChild(option);
                    });
                }
            }
        } catch (error) {
            console.error('Error loading anlasmali kurumlar:', error);
        }
    }

    /**
     * Store original form data
     */
    storeOriginalData() {
        if (this.form) {
            const formData = new FormData(this.form);
            for (const [key, value] of formData.entries()) {
                this.originalData[key] = value;
            }
        }
    }

    /**
     * Setup auto-save functionality
     */
    setupAutoSave() {
        let autoSaveTimeout;
        if (this.form) {
            this.form.addEventListener('input', () => {
                clearTimeout(autoSaveTimeout);
                autoSaveTimeout = setTimeout(() => {
                    this.autoSave();
                }, 2000);
            });
        }
    }

    /**
     * Auto-save form data
     */
    autoSave() {
        if (this.isSubmitting) return;

        const allData = {};
        if (this.form) {
            const formData = new FormData(this.form);
            for (const [key, value] of formData.entries()) {
                allData[key] = value;
            }
        }
        
        // Save to localStorage
        localStorage.setItem('randevuEditAutoSave', JSON.stringify({
            data: allData,
            timestamp: Date.now()
        }));
    }

    /**
     * Setup keyboard shortcuts
     */
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + S for save
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                this.handleSave(e);
            }

            // Ctrl/Cmd + Z for cancel
            if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
                e.preventDefault();
                window.history.back();
            }
        });
    }

    /**
     * Setup time input for 24-hour format
     */
    setupTimeInput() {
        const timeInput = document.getElementById('appointment_time');
        if (!timeInput) return;

        // Add event listener for real-time formatting
        timeInput.addEventListener('input', (e) => {
            let value = e.target.value;
            
            // Remove any non-numeric characters except colon
            value = value.replace(/[^\d:]/g, '');
            
            // Auto-add colon after 2 digits
            if (value.length === 2 && !value.includes(':')) {
                value = value + ':';
            }
            
            // Limit to 5 characters (HH:MM)
            if (value.length > 5) {
                value = value.substring(0, 5);
            }
            
            e.target.value = value;
        });

        // Add event listener for validation and formatting
        timeInput.addEventListener('blur', (e) => {
            const value = e.target.value.trim();
            if (value) {
                // Validate and format the time
                const timeRegex = /^([01]?[0-9]|2[0-3]):[0-5][0-9]$/;
                
                if (timeRegex.test(value)) {
                    // Format with leading zeros
                    const parts = value.split(':');
                    const hours = parts[0].padStart(2, '0');
                    const minutes = parts[1].padStart(2, '0');
                    e.target.value = `${hours}:${minutes}`;
                } else {
                    // Try to fix common issues
                    const parts = value.split(':');
                    if (parts.length === 2) {
                        let hours = parseInt(parts[0]) || 0;
                        let minutes = parseInt(parts[1]) || 0;
                        
                        // Validate hours (0-23)
                        if (hours < 0) hours = 0;
                        if (hours > 23) hours = 23;
                        
                        // Validate minutes (0-59)
                        if (minutes < 0) minutes = 0;
                        if (minutes > 59) minutes = 59;
                        
                        const formattedTime = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`;
                        e.target.value = formattedTime;
                    } else if (value.length === 4 && !value.includes(':')) {
                        // If user typed 4 digits, add colon
                        const hours = value.substring(0, 2);
                        const minutes = value.substring(2, 4);
                        const h = Math.min(23, Math.max(0, parseInt(hours)));
                        const m = Math.min(59, Math.max(0, parseInt(minutes)));
                        e.target.value = `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`;
                    }
                }
            }
        });

        // Add keydown listener for better UX
        timeInput.addEventListener('keydown', (e) => {
            // Allow: backspace, delete, tab, escape, enter, colon
            if ([8, 9, 27, 13, 46, 58].indexOf(e.keyCode) !== -1 ||
                // Allow: Ctrl+A, Ctrl+C, Ctrl+V, Ctrl+X
                (e.keyCode === 65 && e.ctrlKey === true) ||
                (e.keyCode === 67 && e.ctrlKey === true) ||
                (e.keyCode === 86 && e.ctrlKey === true) ||
                (e.keyCode === 88 && e.ctrlKey === true) ||
                // Allow: home, end, left, right, up, down
                (e.keyCode >= 35 && e.keyCode <= 40)) {
                return;
            }
            // Ensure that it is a number and stop the keypress
            if ((e.shiftKey || (e.keyCode < 48 || e.keyCode > 57)) && (e.keyCode < 96 || e.keyCode > 105)) {
                e.preventDefault();
            }
        });
    }

    /**
     * Convert date for backend
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        const toRFC3339WithOffset = (d) => {
            const pad = (n) => String(n).padStart(2, '0');
            const year = d.getFullYear();
            const month = pad(d.getMonth() + 1);
            const day = pad(d.getDate());
            const hours = pad(d.getHours());
            const minutes = pad(d.getMinutes());
            const seconds = pad(d.getSeconds());
            const tzOffsetMin = -d.getTimezoneOffset();
            const sign = tzOffsetMin >= 0 ? '+' : '-';
            const abs = Math.abs(tzOffsetMin);
            const offH = pad(Math.floor(abs / 60));
            const offM = pad(abs % 60);
            return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}${sign}${offH}:${offM}`;
        };

        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            const [y, m, d] = dateValue.split('-').map((v) => parseInt(v));
            const local = new Date(y, (m - 1), d, 0, 0, 0, 0);
            return toRFC3339WithOffset(local);
        }

        if (String(dateValue).includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }

        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return toRFC3339WithOffset(date);
        }

        return '0001-01-01T00:00:00Z';
    }

    /**
     * Convert time for backend
     */
    convertTimeForBackend(timeValue) {
        if (!timeValue || timeValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        const toRFC3339WithOffset = (d) => {
            const pad = (n) => String(n).padStart(2, '0');
            const year = d.getFullYear();
            const month = pad(d.getMonth() + 1);
            const day = pad(d.getDate());
            const hours = pad(d.getHours());
            const minutes = pad(d.getMinutes());
            const seconds = pad(d.getSeconds());
            const tzOffsetMin = -d.getTimezoneOffset();
            const sign = tzOffsetMin >= 0 ? '+' : '-';
            const abs = Math.abs(tzOffsetMin);
            const offH = ("" + Math.floor(abs / 60)).padStart(2, '0');
            const offM = ("" + (abs % 60)).padStart(2, '0');
            return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}${sign}${offH}:${offM}`;
        };

        // If it's already in HH:MM format, convert to full datetime with local offset
        if (/^\d{2}:\d{2}$/.test(timeValue)) {
            const now = new Date();
            const [h, m] = timeValue.split(':');
            now.setHours(parseInt(h), parseInt(m), 0, 0);
            return toRFC3339WithOffset(now);
        }

        const date = new Date(timeValue);
        if (!isNaN(date.getTime())) {
            return toRFC3339WithOffset(date);
        }

        return '0001-01-01T00:00:00Z';
    }

    /**
     * Handle save action
     */
    async handleSave(e) {
        e.preventDefault();

        if (this.isSubmitting) return;

        // Validate all forms
        if (!this.validateAllForms()) {
            this.showAlert('Lütfen formdaki hataları düzeltin.', 'error');
            return;
        }

        this.isSubmitting = true;
        this.setButtonLoading(this.saveBtn, true);

        try {
            // Collect data from the single form
            const allData = {};
            if (this.form) {
                const formData = new FormData(this.form);
                for (const [key, value] of formData.entries()) {
                    if (value && String(value).trim() !== '') {
                        allData[key] = value;
                    }
                }
            }

            // Also collect data from any inputs not in forms
            const allInputs = document.querySelectorAll('input, select, textarea');
            allInputs.forEach(input => {
                if (input.name && input.value && input.value.trim() !== '') {
                    // Handle date/time fields with proper conversion
                    if (input.name === 'appointment_date') {
                        allData[input.name] = this.convertDateForBackend(input.value);
                    } else if (input.name === 'appointment_time') {
                        allData[input.name] = this.convertTimeForBackend(input.value);
                    } else if (input.name === 'old_appointment_date') {
                        allData[input.name] = this.convertDateForBackend(input.value);
                    } else if (input.name === 'old_appointment_time') {
                        allData[input.name] = this.convertTimeForBackend(input.value);
                    } else if (input.name === 'patient_birth_date') {
                        allData[input.name] = this.convertDateForBackend(input.value);
                    } else if (input.name === 'old_patient_birth_date') {
                        allData[input.name] = this.convertDateForBackend(input.value);
					} else if (input.name === 'duration' || input.name === 'old_duration') {
						const numericValue = Number(input.value);
						if (!Number.isNaN(numericValue)) {
							allData[input.name] = numericValue;
						}
					} else if (input.name === 'price' || input.name === 'old_price') {
						const floatValue = parseFloat(String(input.value).replace(',', '.'));
						if (!Number.isNaN(floatValue)) {
							allData[input.name] = floatValue;
						}
					} else if (input.name === 'reminder_sent' || input.name === 'old_reminder_sent') {
						if (input.type === 'checkbox') {
							allData[input.name] = input.checked;
						} else {
							const val = String(input.value).toLowerCase();
							allData[input.name] = (val === 'true' || val === '1' || val === 'on' || val === 'yes');
						}
                    } else {
                        allData[input.name] = input.value;
                    }
                }
            });

            console.log('Sending data:', allData);
            console.log('Randevu ID:', this.randevuData.id);

            const response = await fetch(`/backend/randevu/${this.randevuData.id}/edit`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(allData)
            });

            console.log('Response status:', response.status);
            const result = await response.json();
            console.log('Response data:', result);

            if (result.status === 201) {
                this.showAlert('Randevu başarıyla güncellendi.', 'success');
                this.storeOriginalData();
                localStorage.removeItem('randevuEditAutoSave');
                
                // Redirect after success
                setTimeout(() => {
                    window.location.href = '/panel/randevular';
                }, 1500);
            } else {
                this.showAlert(result.message || 'Randevu güncellenirken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error updating randevu:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.isSubmitting = false;
            this.setButtonLoading(this.saveBtn, false);
        }
    }

    /**
     * Validate all forms
     */
    validateAllForms() {
        if (!this.form) return true;
        let isValid = true;
        const inputs = this.form.querySelectorAll('input, select, textarea');
        inputs.forEach(input => {
            if (!this.validateField(input)) {
                isValid = false;
            }
        });
        return isValid;
    }

    /**
     * Detect form changes
     */
    detectChanges() {
        const hasChanges = this.hasUnsavedChanges();
        
        // Update save button state
        if (this.saveBtn) {
            this.saveBtn.disabled = !hasChanges || this.isSubmitting;
        }
    }

    /**
     * Check if form has unsaved changes
     */
    hasUnsavedChanges() {
        const currentData = {};
        Object.values(this.forms).forEach(form => {
            if (form) {
                const formData = new FormData(form);
                for (const [key, value] of formData.entries()) {
                    currentData[key] = value;
                }
            }
        });

        return Object.keys(currentData).some(key => {
            return currentData[key] !== this.originalData[key];
        });
    }

    /**
     * Validate individual field
     */
    validateField(input) {
        const fieldName = input.name;
        const value = input.value.trim();
        const rules = this.validationRules[fieldName];

        if (!rules) return true;

        let isValid = true;
        let errorMessage = '';

        // Required validation
        if (rules.required && !value) {
            isValid = false;
            errorMessage = `${this.getFieldLabel(fieldName)} gereklidir.`;
        }

        // Length validation
        if (isValid && value && rules.minLength && value.length < rules.minLength) {
            isValid = false;
            errorMessage = `${this.getFieldLabel(fieldName)} en az ${rules.minLength} karakter olmalıdır.`;
        }

        if (isValid && value && rules.maxLength && value.length > rules.maxLength) {
            isValid = false;
            errorMessage = `${this.getFieldLabel(fieldName)} en fazla ${rules.maxLength} karakter olmalıdır.`;
        }

        // Pattern validation
        if (isValid && value && rules.pattern && !rules.pattern.test(value)) {
            isValid = false;
            errorMessage = rules.message || `${this.getFieldLabel(fieldName)} geçerli formatta değil.`;
        }

        // Numeric validation
        if (isValid && value && rules.min !== undefined && parseFloat(value) < rules.min) {
            isValid = false;
            errorMessage = `${this.getFieldLabel(fieldName)} en az ${rules.min} olmalıdır.`;
        }

        if (isValid && value && rules.max !== undefined && parseFloat(value) > rules.max) {
            isValid = false;
            errorMessage = `${this.getFieldLabel(fieldName)} en fazla ${rules.max} olmalıdır.`;
        }

        // Show/hide error
        if (isValid) {
            this.clearFieldError(input);
        } else {
            this.showFieldError(input, errorMessage);
        }

        return isValid;
    }

    /**
     * Show field error
     */
    showFieldError(input, message) {
        input.classList.add('error');
        input.classList.remove('success');

        let errorElement = input.parentNode.querySelector('.field-error');
        if (!errorElement) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            input.parentNode.appendChild(errorElement);
        }

        errorElement.textContent = message;
    }

    /**
     * Clear field error
     */
    clearFieldError(input) {
        input.classList.remove('error');
        input.classList.add('success');

        const errorElement = input.parentNode.querySelector('.field-error');
        if (errorElement) {
            errorElement.textContent = '';
        }
    }

    /**
     * Clear all errors
     */
    clearAllErrors() {
        const inputs = this.form?.querySelectorAll('input, select, textarea');
        inputs?.forEach(input => {
            input.classList.remove('error', 'success');
            const errorElement = input.parentNode.querySelector('.field-error');
            if (errorElement) {
                errorElement.textContent = '';
            }
        });
    }

    /**
     * Get field label
     */
    getFieldLabel(fieldName) {
        const labels = {
            patient_first_name: 'Ad',
            patient_last_name: 'Soyad',
            patient_phone: 'Telefon',
            patient_email: 'E-posta',
            patient_tc_kimlik: 'TC Kimlik',
            patient_birth_date: 'Doğum Tarihi',
            patient_gender: 'Cinsiyet',
            appointment_date: 'Randevu Tarihi',
            appointment_time: 'Randevu Saati',
            duration: 'Süre',
            price: 'Ücret',
            complaint: 'Şikayet',
            notes: 'Notlar',
            cancel_reason: 'İptal Nedeni',
            status: 'Durum',
            payment_status: 'Ödeme Durumu'
        };

        return labels[fieldName] || fieldName;
    }

    /**
     * Detect form changes
     */
    detectChanges() {
        if (!this.form) return;

        const formData = new FormData(this.form);
        const currentData = Object.fromEntries(formData.entries());

        const hasChanges = Object.keys(currentData).some(key => {
            return currentData[key] !== this.originalData[key];
        });

        // Update submit button state
        if (this.submitBtn) {
            this.submitBtn.disabled = !hasChanges || this.isSubmitting;
        }
    }

    /**
     * Check if form has unsaved changes
     */
    hasUnsavedChanges() {
        if (!this.form) return false;

        const formData = new FormData(this.form);
        const currentData = Object.fromEntries(formData.entries());

        return Object.keys(currentData).some(key => {
            return currentData[key] !== this.originalData[key];
        });
    }

    /**
     * Handle before unload
     */
    handleBeforeUnload(e) {
        if (this.hasUnsavedChanges()) {
            e.preventDefault();
            e.returnValue = 'Kaydedilmemiş değişiklikler var. Sayfadan çıkmak istediğinizden emin misiniz?';
            return e.returnValue;
        }
    }

    /**
     * Show field error
     */
    showFieldError(input, message) {
        input.classList.add('error');
        input.classList.remove('success');

        let errorElement = input.parentNode.querySelector('.field-error');
        if (!errorElement) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            input.parentNode.appendChild(errorElement);
        }

        errorElement.textContent = message;
    }

    /**
     * Clear field error
     */
    clearFieldError(input) {
        input.classList.remove('error');
        input.classList.add('success');

        const errorElement = input.parentNode.querySelector('.field-error');
        if (errorElement) {
            errorElement.textContent = '';
        }
    }

    /**
     * Clear all errors
     */
    clearAllErrors() {
        Object.values(this.forms).forEach(form => {
            if (form) {
                const inputs = form.querySelectorAll('input, select, textarea');
                inputs.forEach(input => {
                    input.classList.remove('error', 'success');
                    const errorElement = input.parentNode.querySelector('.field-error');
                    if (errorElement) {
                        errorElement.textContent = '';
                    }
                });
            }
        });
    }

    /**
     * Get field label
     */
    getFieldLabel(fieldName) {
        const labels = {
            patient_first_name: 'Ad',
            patient_last_name: 'Soyad',
            patient_phone: 'Telefon',
            patient_email: 'E-posta',
            patient_tc_kimlik: 'TC Kimlik',
            patient_birth_date: 'Doğum Tarihi',
            patient_gender: 'Cinsiyet',
            appointment_date: 'Randevu Tarihi',
            appointment_time: 'Randevu Saati',
            duration: 'Süre',
            price: 'Ücret',
            complaint: 'Şikayet',
            notes: 'Notlar',
            cancel_reason: 'İptal Nedeni',
            status: 'Durum',
            payment_status: 'Ödeme Durumu'
        };

        return labels[fieldName] || fieldName;
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
        // Remove existing alerts
        const existingAlerts = document.querySelectorAll('.alert');
        existingAlerts.forEach(alert => alert.remove());

        // Create new alert
        const alert = document.createElement('div');
        alert.className = `alert alert-${type}`;
        alert.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 10000;
            padding: 1rem 1.5rem;
            border-radius: 8px;
            color: white;
            font-weight: 500;
            box-shadow: 0 4px 15px rgba(0, 0, 0, 0.2);
            animation: slideIn 0.3s ease-out;
            max-width: 400px;
            word-wrap: break-word;
        `;

        if (type === 'success') {
            alert.style.background = 'linear-gradient(135deg, #28a745 0%, #20c997 100%)';
        } else {
            alert.style.background = 'linear-gradient(135deg, #dc3545 0%, #e74c3c 100%)';
        }

        alert.innerHTML = `
            <div style="display: flex; align-items: center; gap: 0.5rem;">
                <i class="fas fa-${type === 'success' ? 'check-circle' : 'exclamation-circle'}"></i>
                <span>${message}</span>
            </div>
        `;

        document.body.appendChild(alert);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (alert.parentNode) {
                alert.style.animation = 'slideOut 0.3s ease-in';
                setTimeout(() => alert.remove(), 300);
            }
        }, 5000);
    }

    /**
     * Close all modals
     */
    closeModals() {
        const modals = document.querySelectorAll('.modal');
        modals.forEach(modal => {
            modal.style.display = 'none';
        });
    }
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }

    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
`;
document.head.appendChild(style);

// Global functions for HTML onclick attributes
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.style.display = 'none';
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.randevuEditManager = new RandevuEditManager();
});
