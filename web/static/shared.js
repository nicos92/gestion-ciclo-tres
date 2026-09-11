function togglePassword(fieldId, iconId) {
    var passwordField = document.getElementById(fieldId);
    var icon = document.getElementById(iconId);

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
    var password = document.getElementById('password') || document.getElementById('newPassword');
    var confirmPassword = document.getElementById('confirmPassword');
    if (!password || !confirmPassword) return;

    var val = confirmPassword.value;
    var messageDiv = document.getElementById('passwordMatchMessage');
    var messageText = document.getElementById('passwordMatchText');
    var passwordDivMessage = document.getElementById('passwordDivMessage');

    if (!messageDiv || !messageText || !passwordDivMessage) return;

    if (val.length > 0) {
        if (password.value === val) {
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

var tarimaIdPendiente = null;

function confirmarEliminar(id) {
    tarimaIdPendiente = id;
    var modal = new bootstrap.Modal(document.getElementById('modalEliminar'));
    modal.show();
}

document.addEventListener('DOMContentLoaded', function() {
    var btnConfirmar = document.getElementById('btnConfirmarEliminar');
    if (btnConfirmar) {
        btnConfirmar.addEventListener('click', function() {
            if (tarimaIdPendiente !== null) {
                var modal = bootstrap.Modal.getInstance(document.getElementById('modalEliminar'));
                if (modal) modal.hide();
                htmx.ajax('DELETE', '/tarimas/' + tarimaIdPendiente, {
                    target: '#tarima-' + tarimaIdPendiente,
                    swap: 'delete'
                });
                tarimaIdPendiente = null;
            }
        });
    }

    var modalEl = document.getElementById('modalEliminar');
    if (modalEl) {
        modalEl.addEventListener('hidden.bs.modal', function() {
            tarimaIdPendiente = null;
        });
    }

    var password = document.getElementById('password');
    var newPassword = document.getElementById('newPassword');
    var confirmPassword = document.getElementById('confirmPassword');
    if (password) password.addEventListener('input', checkPasswordMatch);
    if (newPassword) newPassword.addEventListener('input', checkPasswordMatch);
    if (confirmPassword) confirmPassword.addEventListener('input', checkPasswordMatch);

    autoDismissAlert();
    document.body.addEventListener('htmx:afterSwap', function (evt) {
        if (evt.detail.target && evt.detail.target.id === 'form-container') {
            autoDismissAlert();
            var codigoBarras = document.getElementById('codigoBarras');
            if (codigoBarras) codigoBarras.focus();
        }
    });
});


function autoDismissAlert() {
    var alert = document.getElementById('auto-dismiss-alert');
    if (alert) {
        setTimeout(function () {
            var bsAlert = bootstrap.Alert.getInstance(alert) || new bootstrap.Alert(alert);
            bsAlert.close();
        }, 5000);
    }
}

function limpiarFiltros() {
    var ids = [
        'numero_producto', 'numero_tarima', 'numero_usuario', 'numero_venta',
        'fecha_registro', 'legajo', 'nombre_usuario',
        'cantidad_cajas_min', 'peso_min'
    ];
    ids.forEach(function(id) {
        var el = document.getElementById(id);
        if (el) el.value = '';
    });
    if (window.htmx) {
        htmx.trigger('#filtroTarimas', 'submit');
    } else {
        document.getElementById('filtroTarimas').submit();
    }
}

function autoFillFromBarcode(input) {
    var value = input.value.toString();
    value = value.replace(/[^0-9]/g, '');
    if (value.length > 30) {
        value = value.substring(0, 30);
    }
    input.value = value;

    var invalid0 = document.getElementById('invalidFormat0');
    var invalid9998 = document.getElementById('invalidFormat9998');

  var hideErrors = function () {
        input.classList.remove('is-invalid');
        if (invalid0) invalid0.style.display = 'none';
        if (invalid9998) invalid9998.style.display = 'none';
  };

    if (value.length < 30) {
        hideErrors();
        return;
    }

    if (value.length === 30) {
        if (value.charAt(0) !== '0') {
            input.classList.add('is-invalid');
            if (invalid0) invalid0.style.display = 'block';
            if (invalid9998) invalid9998.style.display = 'none';
            return;
        }
        if (value.substring(13, 17) !== '9998') {
            input.classList.add('is-invalid');
            if (invalid0) invalid0.style.display = 'none';
            if (invalid9998) invalid9998.style.display = 'block';
            return;
        }
        hideErrors();

        var producto = document.getElementById('numeroProducto');
        var tarima = document.getElementById('numeroTarima');
        var usuario = document.getElementById('numeroUsuario');
        var conservacion = document.getElementById('conservacion');
        var cajas = document.getElementById('cantidadCajas');
        var peso = document.getElementById('peso');
        var venta = document.getElementById('numeroVenta');

        if(producto) producto.value = value.substring(1, 7);
        if (tarima) tarima.value = value.substring(7, 13);
        if (conservacion) conservacion.value = value.substring(17, 18);
        if (usuario) usuario.value = value.substring(18, 21);
        if (cajas) cajas.value = parseInt(value.substring(21, 24), 10);

        var lastSix = value.substring(24, 30);
        if (lastSix.length === 6 && peso) {
            var wholePart = lastSix.substring(0, 4);
            var decimalPart = lastSix.substring(4, 6);
            peso.value = wholePart + '.' + decimalPart;
        }

        if (venta) {
            if (!venta.value.trim()) {
                var currentYear = new Date().getFullYear().toString().substr(-2);
                venta.value = currentYear + '-';
            }
            venta.focus();
        }
    }
}
