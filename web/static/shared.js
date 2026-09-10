function togglePassword(fieldId, iconId) {
    const passwordField = document.getElementById(fieldId);
    const icon = document.getElementById(iconId);

    if (passwordField.type === 'password') {
        passwordField.type = 'text';
        icon.classList.remove('fa-eye-slash');
        icon.classList.add('fa-eye');
    } else {
        passwordField.type = 'password';
        icon.classList.remove('fa-eye');
        icon.classList.add('fa-eye-slash');
    }
}

function checkPasswordMatch() {
    const password = document.getElementById('password').value;
    const confirmPassword = document.getElementById('confirmPassword');
    if (!confirmPassword) return;

    const val = confirmPassword.value;
    const messageDiv = document.getElementById('passwordMatchMessage');
    const messageText = document.getElementById('passwordMatchText');
    const passwordDivMessage = document.getElementById('passwordDivMessage');

    if (!messageDiv || !messageText || !passwordDivMessage) return;

    if (val.length > 0) {
        if (password === val) {
            passwordDivMessage.className = 'mb-0 p-2 alert alert-success';
            messageText.innerHTML = '<i class="fas fa-check-circle text-success me-2"></i>Las contraseñas coinciden';
            messageDiv.className = 'mb-0 p-0 alert alert-success';
            messageDiv.style.display = 'block';
        } else {
            messageText.innerHTML = '<i class="fas fa-times-circle text-danger me-2"></i>Las contraseñas no coinciden';
            messageDiv.className = 'mb-0 p-0 alert alert-danger';
            messageDiv.style.display = 'block';
            passwordDivMessage.className = 'mb-0 alert alert-danger';
        }
    } else {
        messageDiv.style.display = 'none';
    }
}

document.addEventListener('DOMContentLoaded', function() {
    const password = document.getElementById('password');
    const confirmPassword = document.getElementById('confirmPassword');
    if (password) password.addEventListener('input', checkPasswordMatch);
    if (confirmPassword) confirmPassword.addEventListener('input', checkPasswordMatch);
});
