/**
 * Randevu Talebi (Individual Appointment Request View) Page JavaScript
 * Handles appointment creation, delete functionality, and AJAX operations
 */

class RandevuTalebiViewManager {
    constructor() {
        this.randevuTalebiData = window.randevuTalebiData || {};
        this.currentSubeId = null;
        this.formData = {
            subeler: [],
            branslar: [],
            doktorlar: [],
            anlasmaliKurumlar: [],
            tedkikler: [],
            tibbiBirimler: []
        };
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupDeleteButton();
        this.setupCreateAppointmentButton();
        this.setupViewRelatedAppointmentButton();
        this.setupToggleStatusButton();
        this.setupTimeInput();
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
                const createModal = document.getElementById('createAppointmentModal');

                if (deleteModal && deleteModal.style.display !== 'none') {
                    this.closeDeleteModal();
                } else if (createModal && createModal.style.display !== 'none') {
                    this.closeCreateAppointmentModal();
                }
            }
        });

        // Close modals when clicking outside
        document.addEventListener('click', (e) => {
            const deleteModal = document.getElementById('deleteModal');
            const createModal = document.getElementById('createAppointmentModal');

            if (e.target === deleteModal) {
                this.closeDeleteModal();
            } else if (e.target === createModal) {
                this.closeCreateAppointmentModal();
            }
        });
    }

    /**
     * Setup delete button functionality
     */
    setupDeleteButton() {
        const deleteBtn = document.getElementById('deleteRequestBtn');
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
                this.deleteRequest();
            });
        }
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
     * Setup create appointment button
     */
    setupCreateAppointmentButton() {
        const createBtn = document.getElementById('createAppointmentBtn');
        if (!createBtn) return;

        createBtn.addEventListener('click', (e) => {
            e.preventDefault();
            this.showCreateAppointmentModal();
        });

        // Setup save appointment button
        const saveBtn = document.getElementById('saveAppointmentBtn');
        if (saveBtn) {
            saveBtn.addEventListener('click', (e) => {
                e.preventDefault();
                this.saveAppointment();
            });
        }

        // Setup form change handlers
        this.setupFormHandlers();
    }

    /**
     * Setup view related appointment button
     */
    setupViewRelatedAppointmentButton() {
        const viewBtn = document.getElementById('viewRelatedAppointmentBtn');
        if (!viewBtn) return;

        viewBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const rid = viewBtn.getAttribute('data-rid');
            if (rid) {
                window.location.href = `/panel/randevular/${rid}`;
            }
        });
    }

    /**
     * Setup toggle status button
     */
    setupToggleStatusButton() {
        const toggleBtn = document.getElementById('toggleStatusBtn');
        if (!toggleBtn) return;

        toggleBtn.addEventListener('click', (e) => {
            e.preventDefault();
            const currentStatus = toggleBtn.getAttribute('data-status');
            this.toggleStatus(currentStatus, toggleBtn);
        });
    }

    /**
     * Get status text in Turkish
     */
    getStatusText(status) {
        const statusMap = {
            'yeni': 'Yeni',
            'randevu-verildi': 'Randevu Verildi',
            'randevu-verilemedi': 'Randevu Verilemedi',
            'ulasilamadi': 'Ulaşılamadı',
            'gelmedi': 'Gelmedi',
            'hasta-vazgecti': 'Hasta Vazgeçti'
        };
        return statusMap[status] || status;
    }

    /**
     * Toggle status
     */
    async toggleStatus(currentStatus, button) {
        if (!currentStatus || !this.randevuTalebiData.rrid) return;

        const statusBadge = document.getElementById('statusBadge');
        if (!statusBadge) return;

        const originalText = button.querySelector('.btn-text').textContent;
        const icon = button.querySelector('i');
        const originalIconClass = icon.className;

        try {
            // Show loading state
            this.setButtonLoading(button, true);
            icon.className = 'fas fa-spinner fa-spin';

            const response = await fetch(`/backend/randevu-request/${this.randevuTalebiData.rrid}/toggle-status`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Requested-With': 'XMLHttpRequest'
                },
                body: JSON.stringify({
                    status: currentStatus
                })
            });

            const responseData = await response.json().catch(() => ({}));

            if (responseData.status === 201 && responseData.new_status) {
                const newStatus = responseData.new_status;

                // Update status badge
                statusBadge.className = `status-badge status-badge--${newStatus}`;
                statusBadge.setAttribute('data-status', newStatus);
                statusBadge.textContent = this.getStatusText(newStatus);

                // Update button data-status attribute
                button.setAttribute('data-status', newStatus);

                // Update data
                this.randevuTalebiData.status = newStatus;

                this.showAlert('Durum başarıyla güncellendi.', 'success');
            } else {
                this.showAlert(responseData.message || 'Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            }
        } catch (error) {
            console.error('Toggle status error:', error);
            this.showAlert('Bağlantı hatası oluştu.', 'error');
        } finally {
            // Reset button state
            this.setButtonLoading(button, false);
            icon.className = originalIconClass;
        }
    }

    /**
     * Setup form handlers for cascading dropdowns
     */
    setupFormHandlers() {
        const sidSelect = document.getElementById('sid');
        const bridSelect = document.getElementById('brid');
        const dridSelect = document.getElementById('drid');

        if (sidSelect) {
            sidSelect.addEventListener('change', async (e) => {
                const sid = e.target.value;
                this.currentSubeId = sid;
                
                if (sid) {
                    // Enable and load branches
                    bridSelect.disabled = false;
                    dridSelect.disabled = false;
                    await this.loadBranches(sid);
                    await this.loadDoctors(sid);
                } else {
                    // Reset dependent fields
                    bridSelect.disabled = true;
                    bridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                    dridSelect.disabled = true;
                    dridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
                }
            });
        }
    }

    /**
     * Show create appointment modal
     */
    async showCreateAppointmentModal() {
        const modal = document.getElementById('createAppointmentModal');
        if (!modal) return;

        // Load initial data
        await this.loadInitialFormData();

        modal.style.display = 'flex';
        document.body.style.overflow = 'hidden';

        // Pre-fill sid if available
        if (this.randevuTalebiData.sid) {
            const sidSelect = document.getElementById('sid');
            if (sidSelect) {
                sidSelect.value = this.randevuTalebiData.sid;
                this.currentSubeId = this.randevuTalebiData.sid;
                await this.loadBranches(this.randevuTalebiData.sid);
                await this.loadDoctors(this.randevuTalebiData.sid);
                
                // Pre-fill drid if available
                if (this.randevuTalebiData.drid) {
                    const dridSelect = document.getElementById('drid');
                    if (dridSelect) {
                        dridSelect.value = this.randevuTalebiData.drid;
                    }
                }
            }
        }
    }

    /**
     * Close create appointment modal
     */
    closeCreateAppointmentModal() {
        const modal = document.getElementById('createAppointmentModal');
        if (!modal) return;
        
        modal.style.display = 'none';
        document.body.style.overflow = '';
        this.resetAppointmentForm();
    }

    /**
     * Load initial form data (subeler, anlasmali kurumlar, tedkikler, tibbi birimler)
     */
    async loadInitialFormData() {
        try {
            // Load all in parallel
            await Promise.all([
                this.loadSubeler(),
                this.loadAnlasmaliKurumlar(),
                this.loadTedkikler(),
                this.loadTibbiBirimler()
            ]);
        } catch (error) {
            console.error('Error loading initial form data:', error);
            this.showAlert('Form verileri yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load subeler
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
            
            if (data.status === 200) {
                this.formData.subeler = data.data;
                select.innerHTML = '<option value="">Şube seçiniz...</option>';
                
                data.data.forEach(sube => {
                    const option = document.createElement('option');
                    option.value = sube.sid;
                    option.textContent = `${sube.name} - ${sube.city}`;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading subeler:', error);
            this.showAlert('Şubeler yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load branches for selected sube
     */
    async loadBranches(sid) {
        const select = document.getElementById('brid');
        if (!select) return;

        try {
            const response = await fetch(`/backend/sube/${sid}/get-branches`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.formData.branslar = data.data;
                select.innerHTML = '<option value="">Branş seçiniz...</option>';
                
                data.data.forEach(brans => {
                    const option = document.createElement('option');
                    option.value = brans.brid;
                    option.textContent = brans.name;
                    select.appendChild(option);
                });
                
                select.disabled = false;
            }
        } catch (error) {
            console.error('Error loading branches:', error);
            this.showAlert('Branşlar yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load doctors for selected sube
     */
    async loadDoctors(sid) {
        const select = document.getElementById('drid');
        if (!select) return;

        try {
            const response = await fetch(`/backend/sube/${sid}/get-doctors`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.formData.doktorlar = data.data;
                select.innerHTML = '<option value="">Doktor seçiniz...</option>';
                
                data.data.forEach(doktor => {
                    const option = document.createElement('option');
                    option.value = doktor.drid;
                    option.textContent = `${doktor.title} ${doktor.first_name} ${doktor.last_name}`;
                    select.appendChild(option);
                });
                
                select.disabled = false;
            }
        } catch (error) {
            console.error('Error loading doctors:', error);
            this.showAlert('Doktorlar yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load anlasmali kurumlar
     */
    async loadAnlasmaliKurumlar() {
        const select = document.getElementById('akid');
        if (!select) return;

        try {
            const response = await fetch('/backend/get-all-anlasmali-kurumlar', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.formData.anlasmaliKurumlar = data.data;
                select.innerHTML = '<option value="">Anlaşmalı kurum seçiniz...</option>';
                
                data.data.forEach(kurum => {
                    const option = document.createElement('option');
                    option.value = kurum.akid;
                    option.textContent = kurum.name;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading anlasmali kurumlar:', error);
            this.showAlert('Anlaşmalı kurumlar yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load tedkikler
     */
    async loadTedkikler() {
        const select = document.getElementById('tid');
        if (!select) return;

        try {
            const response = await fetch('/backend/get-all-tedkikler', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.formData.tedkikler = data.data;
                select.innerHTML = '<option value="">Tetkik seçiniz...</option>';
                
                data.data.forEach(tedkik => {
                    const option = document.createElement('option');
                    option.value = tedkik.tid;
                    option.textContent = tedkik.name;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading tedkikler:', error);
            this.showAlert('Tetkikler yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Load tibbi birimler
     */
    async loadTibbiBirimler() {
        const select = document.getElementById('tbid');
        if (!select) return;

        try {
            const response = await fetch('/backend/get-all-tibbi-birimler', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 200) {
                this.formData.tibbiBirimler = data.data;
                select.innerHTML = '<option value="">Tıbbi birim seçiniz...</option>';
                
                data.data.forEach(birim => {
                    const option = document.createElement('option');
                    option.value = birim.tbid;
                    option.textContent = birim.name;
                    select.appendChild(option);
                });
            }
        } catch (error) {
            console.error('Error loading tibbi birimler:', error);
            this.showAlert('Tıbbi birimler yüklenirken hata oluştu.', 'error');
        }
    }

    /**
     * Convert date for backend submission (ISO or Go zero time)
     */
    convertDateForBackend(dateValue) {
        if (!dateValue || dateValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        if (/^\d{4}-\d{2}-\d{2}$/.test(dateValue)) {
            return dateValue + 'T00:00:00Z';
        }

        if (String(dateValue).includes('00:00, 01/01/0001')) {
            return '0001-01-01T00:00:00Z';
        }

        const date = new Date(dateValue);
        if (!isNaN(date.getTime())) {
            return date.toISOString();
        }

        return '0001-01-01T00:00:00Z';
    }

    /**
     * Convert time for backend submission
     */
    convertTimeForBackend(timeValue) {
        if (!timeValue || timeValue === '') {
            return '0001-01-01T00:00:00Z';
        }

        // If it's in HH:MM format, convert to ISO
        if (/^\d{2}:\d{2}$/.test(timeValue)) {
            return `0001-01-01T${timeValue}:00Z`;
        }

        return '0001-01-01T00:00:00Z';
    }

    /**
     * Save appointment
     */
    async saveAppointment() {
        const form = document.getElementById('createAppointmentForm');
        const saveBtn = document.getElementById('saveAppointmentBtn');
        
        if (!form || !saveBtn) return;

        // Clear previous errors
        this.clearFormErrors(form);

        // Validate required fields
        const requiredFields = [
            { name: 'patient_first_name', label: 'Ad' },
            { name: 'patient_last_name', label: 'Soyad' },
            { name: 'patient_phone', label: 'Telefon' },
            { name: 'sid', label: 'Şube' },
            { name: 'appointment_date', label: 'Randevu Tarihi' },
            { name: 'appointment_time', label: 'Randevu Saati' }
        ];
        
        let isValid = true;
        let firstInvalidField = null;
        let errorMessages = [];
        
        for (const field of requiredFields) {
            const input = form.querySelector(`[name="${field.name}"]`);
            if (input && !input.value.trim()) {
                isValid = false;
                this.markFieldAsInvalid(input, `${field.label} zorunludur`);
                errorMessages.push(field.label);
                
                if (!firstInvalidField) {
                    firstInvalidField = input;
                }
            }
        }

        // Validate email format if provided
        const emailField = form.querySelector('[name="patient_email"]');
        if (emailField && emailField.value.trim()) {
            const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailPattern.test(emailField.value.trim())) {
                isValid = false;
                this.markFieldAsInvalid(emailField, 'Geçerli bir e-posta adresi giriniz');
                errorMessages.push('E-posta (geçersiz format)');
                
                if (!firstInvalidField) {
                    firstInvalidField = emailField;
                }
            }
        }

        // Validate TC Kimlik format if provided
        const tcField = form.querySelector('[name="patient_tc_kimlik"]');
        if (tcField && tcField.value.trim()) {
            const tcPattern = /^[0-9]{11}$/;
            if (!tcPattern.test(tcField.value.trim())) {
                isValid = false;
                this.markFieldAsInvalid(tcField, 'TC Kimlik 11 haneli olmalıdır');
                errorMessages.push('TC Kimlik (geçersiz format)');
                
                if (!firstInvalidField) {
                    firstInvalidField = tcField;
                }
            }
        }

        // Validate time format (24-hour)
        const timeField = form.querySelector('[name="appointment_time"]');
        if (timeField && timeField.value.trim()) {
            const timePattern = /^([01]?[0-9]|2[0-3]):[0-5][0-9]$/;
            if (!timePattern.test(timeField.value.trim())) {
                isValid = false;
                this.markFieldAsInvalid(timeField, 'Geçerli bir saat giriniz (HH:MM formatında)');
                errorMessages.push('Randevu Saati (geçersiz format)');
                
                if (!firstInvalidField) {
                    firstInvalidField = timeField;
                }
            }
        }

        if (!isValid) {
            // Scroll to first invalid field
            if (firstInvalidField) {
                firstInvalidField.scrollIntoView({ behavior: 'smooth', block: 'center' });
                firstInvalidField.focus();
            }
            
            // Show detailed error message
            const errorMsg = errorMessages.length > 3 
                ? `Lütfen zorunlu alanları doldurun (${errorMessages.length} alan eksik)`
                : `Lütfen şu alanları kontrol edin: ${errorMessages.join(', ')}`;
            
            this.showAlert(errorMsg, 'error');
            return;
        }

        this.setButtonLoading(saveBtn, true);
        
        try {
            const formData = new FormData(form);
            const data = {};
            
            // Convert FormData to object
            for (const [key, value] of formData.entries()) {
                if (value === '') {
                    data[key] = null;
                } else {
                    data[key] = value;
                }
            }

            // Convert dates and times
            data.appointment_date = this.convertDateForBackend(data.appointment_date);
            data.appointment_time = this.convertTimeForBackend(data.appointment_time);
            data.patient_birth_date = data.patient_birth_date ? this.convertDateForBackend(data.patient_birth_date) : null;

            // Convert numeric fields
            if (data.duration) data.duration = parseInt(data.duration);
            if (data.price) data.price = parseFloat(data.price);

            // Convert empty string values to null for foreign keys
            const nullableFields = ['drid', 'brid', 'akid', 'tid', 'tbid', 'patient_email', 'patient_tc_kimlik', 'patient_birth_date', 'patient_gender', 'complaint', 'notes'];
            nullableFields.forEach(field => {
                if (data[field] === '' || data[field] === null) {
                    delete data[field];
                }
            });

            const response = await fetch('/backend/add-randevu', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });
            
            const result = await response.json();
            
            if (result.status === 201) {
                this.showAlert('Randevu başarıyla oluşturuldu.', 'success');
                this.closeCreateAppointmentModal();
                setTimeout(() => {
                    window.location.reload();
                }, 1500);
            } else {
                this.showAlert(result.message || 'Randevu oluşturulurken hata oluştu.', 'error');
            }
        } catch (error) {
            console.error('Error saving appointment:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
        } finally {
            this.setButtonLoading(saveBtn, false);
        }
    }

    /**
     * Show delete confirmation modal
     */
    showDeleteModal() {
        const modal = document.getElementById('deleteModal');
        const nameSpan = document.getElementById('deleteItemName');
        
        if (!modal) return;
        
        if (nameSpan) {
            nameSpan.textContent = this.randevuTalebiData.patientName || 'Bu randevu talebi';
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
     * Delete request
     */
    async deleteRequest() {
        const confirmBtn = document.getElementById('confirmDeleteBtn');
        if (!confirmBtn) return;
        
        this.setButtonLoading(confirmBtn, true);
        
        try {
            const response = await fetch(`/backend/randevu-request/${this.randevuTalebiData.rrid}/delete`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const data = await response.json();
            
            if (data.status === 201) {
                this.showAlert('Randevu talebi başarıyla silindi.', 'success');
                setTimeout(() => {
                    window.location.href = '/panel/randevu-talepleri';
                }, 1500);
            } else {
                this.showAlert(data.message || 'Randevu talebi silinirken hata oluştu.', 'error');
                this.setButtonLoading(confirmBtn, false);
            }
        } catch (error) {
            console.error('Error deleting request:', error);
            this.showAlert('Server Hatası: Lütfen daha sonra tekrar deneyin.', 'error');
            this.setButtonLoading(confirmBtn, false);
        }
    }

    /**
     * Reset appointment form
     */
    resetAppointmentForm() {
        const form = document.getElementById('createAppointmentForm');
        if (form) {
            form.reset();
        }
        
        // Reset cascading dropdowns
        const bridSelect = document.getElementById('brid');
        const dridSelect = document.getElementById('drid');
        
        if (bridSelect) {
            bridSelect.disabled = true;
            bridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
        }
        
        if (dridSelect) {
            dridSelect.disabled = true;
            dridSelect.innerHTML = '<option value="">Önce şube seçiniz...</option>';
        }
        
        this.currentSubeId = null;
    }

    /**
     * Mark field as invalid with error message
     */
    markFieldAsInvalid(field, errorMessage) {
        if (!field) return;
        
        // Add error class to field
        field.classList.add('field-error');
        field.style.borderColor = '#dc3545';
        field.style.borderWidth = '2px';
        field.style.boxShadow = '0 0 0 3px rgba(220, 53, 69, 0.15)';
        
        // Find or create error message element
        const formGroup = field.closest('.form-group');
        if (formGroup) {
            let errorElement = formGroup.querySelector('.error-message');
            if (!errorElement) {
                errorElement = document.createElement('div');
                errorElement.className = 'error-message';
                formGroup.appendChild(errorElement);
            }
            errorElement.textContent = errorMessage;
            errorElement.style.display = 'block';
        }
        
        // Add shake animation
        field.style.animation = 'shake 0.5s';
        setTimeout(() => {
            field.style.animation = '';
        }, 500);
        
        // Clear error on input
        const clearError = () => {
            field.classList.remove('field-error');
            field.style.borderColor = '';
            field.style.borderWidth = '';
            field.style.boxShadow = '';
            
            const formGroup = field.closest('.form-group');
            if (formGroup) {
                const errorElement = formGroup.querySelector('.error-message');
                if (errorElement) {
                    errorElement.style.display = 'none';
                }
            }
            
            field.removeEventListener('input', clearError);
            field.removeEventListener('change', clearError);
        };
        
        field.addEventListener('input', clearError);
        field.addEventListener('change', clearError);
    }

    /**
     * Clear all form errors
     */
    clearFormErrors(form) {
        if (!form) return;
        
        // Remove error classes and styles from all fields
        const errorFields = form.querySelectorAll('.field-error');
        errorFields.forEach(field => {
            field.classList.remove('field-error');
            field.style.borderColor = '';
            field.style.borderWidth = '';
            field.style.boxShadow = '';
        });
        
        // Hide all error messages
        const errorMessages = form.querySelectorAll('.error-message');
        errorMessages.forEach(msg => {
            msg.style.display = 'none';
        });
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
            
            // Ctrl/Cmd + N for new appointment
            if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
                e.preventDefault();
                const createBtn = document.getElementById('createAppointmentBtn');
                if (createBtn && !this.randevuTalebiData.hasRelatedAppointment) {
                    this.showCreateAppointmentModal();
                }
            }
        });
    }
}

// Global functions for HTML onclick attributes
function closeCreateAppointmentModal() {
    if (window.randevuTalebiViewManager) {
        window.randevuTalebiViewManager.closeCreateAppointmentModal();
    }
}

function closeDeleteModal() {
    if (window.randevuTalebiViewManager) {
        window.randevuTalebiViewManager.closeDeleteModal();
    }
}

function closeAlert(alertId) {
    if (window.randevuTalebiViewManager) {
        window.randevuTalebiViewManager.closeAlert(alertId);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.randevuTalebiViewManager = new RandevuTalebiViewManager();
});

