// Doktorlar Page JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Initialize WOW.js animations if available
    if (typeof WOW !== 'undefined') {
        new WOW().init();
    }
    
    
    // Enhanced hover effects for team cards
    const teamCards = document.querySelectorAll('.team-card-three');
    
    teamCards.forEach(card => {
        // Add smooth hover effects
        card.addEventListener('mouseenter', function() {
            this.style.transform = 'translateY(-8px)';
            this.style.boxShadow = '0 12px 24px rgba(0, 0, 0, 0.15)';
        });
        
        card.addEventListener('mouseleave', function() {
            this.style.transform = 'translateY(0)';
            this.style.boxShadow = '0 4px 6px rgba(0, 0, 0, 0.1)';
        });
        
        // Add click effect
        card.addEventListener('click', function(e) {
            // Only trigger if not clicking on a link
            if (!e.target.closest('a')) {
                const doctorLink = this.querySelector('.team-card-three__name a');
                if (doctorLink) {
                    doctorLink.click();
                }
            }
        });
    });
    
    // Social media link enhancements
    const socialLinks = document.querySelectorAll('.social-links a');
    
    socialLinks.forEach(link => {
        link.addEventListener('mouseenter', function() {
            this.style.transform = 'scale(1.1)';
        });
        
        link.addEventListener('mouseleave', function() {
            this.style.transform = 'scale(1)';
        });
    });
    
    // Add loading animation for images
    const doctorImages = document.querySelectorAll('.team-card-three__image');
    
    doctorImages.forEach(image => {
        const img = new Image();
        const bgImage = image.style.backgroundImage;
        const url = bgImage.replace(/^url\(['"]?/, '').replace(/['"]?\)$/, '');
        
        img.onload = function() {
            image.style.opacity = '1';
        };
        
        img.onerror = function() {
            // Fallback to default image
            image.style.backgroundImage = 'url(files/defaults/doktorlar/default-doctor-pic.png)';
            image.style.opacity = '1';
        };
        
        if (url && url !== 'none') {
            image.style.opacity = '0';
            image.style.transition = 'opacity 0.3s ease';
            img.src = url;
        }
    });
    
    // Add intersection observer for scroll animations
    if ('IntersectionObserver' in window) {
        const observerOptions = {
            threshold: 0.1,
            rootMargin: '0px 0px -50px 0px'
        };
        
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.style.animationDelay = '0ms';
                    entry.target.classList.add('animate-in');
                }
            });
        }, observerOptions);
        
        teamCards.forEach((card, index) => {
            card.style.animationDelay = `${index * 100}ms`;
            observer.observe(card);
        });
    }
    
    // Add keyboard navigation support
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Enter' || e.key === ' ') {
            const focusedCard = document.activeElement.closest('.team-card-three');
            if (focusedCard) {
                e.preventDefault();
                const doctorLink = focusedCard.querySelector('.team-card-three__name a');
                if (doctorLink) {
                    doctorLink.click();
                }
            }
        }
    });
    
    // Add focus styles for accessibility
    teamCards.forEach(card => {
        card.setAttribute('tabindex', '0');
        card.setAttribute('role', 'button');
        card.setAttribute('aria-label', 'View doctor details');
    });
    
    // Enhanced empty state interaction
    const emptyState = document.querySelector('.empty-state');
    if (emptyState) {
        const button = emptyState.querySelector('.mediox-btn');
        if (button) {
            button.addEventListener('mouseenter', function() {
                this.style.transform = 'translateY(-2px)';
                this.style.boxShadow = '0 8px 16px rgba(0, 0, 0, 0.15)';
            });
            
            button.addEventListener('mouseleave', function() {
                this.style.transform = 'translateY(0)';
                this.style.boxShadow = '0 4px 6px rgba(0, 0, 0, 0.1)';
            });
        }
    }
});

// Add CSS for animation classes
const style = document.createElement('style');
style.textContent = `
    .team-card-three.animate-in {
        animation: fadeInUp 0.6s ease-out forwards;
    }
    
    .team-card-three:focus {
        outline: 2px solid var(--theme-primary-color);
        outline-offset: 4px;
    }
    
    .team-card-three:focus .team-card-three__hover {
        opacity: 1;
    }
`;
document.head.appendChild(style);
