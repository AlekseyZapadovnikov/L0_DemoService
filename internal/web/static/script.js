document.addEventListener('DOMContentLoaded', () => {
    // Находим элементы
    const btnFetch = document.getElementById('btn-fetch');
    const btnClear = document.getElementById('btn-clear');
    const inputUid = document.getElementById('uid');
    const output = document.getElementById('output');

    // Хендлер поиска
    btnFetch.addEventListener('click', () => {
        const uid = inputUid.value.trim();
        if (!uid) {
            alert('Введите UID заказа');
            return;
        }

        fetch(`/order/${uid}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Order not found');
                }
                return response.json();
            })
            .then(data => {
                output.textContent = JSON.stringify(data, null, 2);
            })
            .catch(error => {
                output.textContent = `Ошибка: ${error.message}`;
            });
    });

    // Хендлер очистки
    btnClear.addEventListener('click', () => {
        inputUid.value = '';
        output.textContent = '';
    });
});