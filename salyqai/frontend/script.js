// --- DOM Элементы ---
const chatContainer = document.getElementById('chat-container');
const chatInputArea = document.getElementById('chat-input-area'); // Область с инпутом и кнопкой
const chatInput = document.getElementById('chat-input');
const sendChatBtn = document.getElementById('send-chat-btn');
const loadingChatBtn = document.getElementById('loading-chat-btn'); // Кнопка загрузки
const interactiveArea = document.getElementById('interactive-area'); // Место для формы
const errorSection = document.getElementById('error-section');
const errorMessage = document.getElementById('error-message');
const disclaimerSection = document.getElementById('disclaimer-section');
const disclaimerDiv = document.getElementById('disclaimer');

// Шаблоны
const formTemplate = document.getElementById('calculation-form-template');
const resultTemplate = document.getElementById('calculation-result-template');

// --- API URL ---
const CHAT_API_URL = 'http://localhost:8080/api/v1/chat';
const CALC_API_URL = 'http://localhost:8080/api/v1/calculate_from_form';

// --- Состояние ---
let isWaitingForAi = false; // Флаг ожидания ответа от AI
let disclaimerShown = false; // Показан ли дисклеймер

// --- Инициализация ---
window.onload = () => {
    // Можно добавить стартовое сообщение или оставить пустым
    addMessageToChat('ai', 'Здравствуйте! Я SalyqAI. Чем могу помочь сегодня по налогам ИП на Упрощенке?');
};

// --- Обработчики ввода в чате ---
sendChatBtn.addEventListener('click', sendChatMessage);
chatInput.addEventListener('keypress', (event) => {
    if (event.key === 'Enter' && !isWaitingForAi) {
        sendChatMessage();
    }
});

// --- Отправка сообщения в чат ---
async function sendChatMessage() {
    const userMessage = chatInput.value.trim();
    if (!userMessage || isWaitingForAi) return;

    addMessageToChat('user', userMessage);
    chatInput.value = '';
    setChatLoading(true); // Показываем индикатор загрузки
    hideError(); // Скрываем старые ошибки API
    removeEmbeddedForm(); // Убираем форму, если она была

    try {
        const response = await fetch(CHAT_API_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message: userMessage /*, history: [] - можно добавить историю */ }),
        });

        if (!response.ok) {
            let errorDetails = `HTTP status: ${response.status}`;
            try {
                const errorData = await response.json();
                errorDetails += ` - ${errorData.error || errorData.message || JSON.stringify(errorData)}`;
            } catch(e) { /* ignore */ }
            throw new Error(`Ошибка сети или сервера: ${errorDetails}`);
        }

        const data = await response.json();
        handleApiResponse(data);

    } catch (error) {
        console.error("Chat API Error:", error);
        showError(`Ошибка при общении с AI: ${error.message}`);
        addMessageToChat('ai', 'Извините, произошла ошибка. Попробуйте еще раз позже.');
    } finally {
        setChatLoading(false); // Убираем индикатор загрузки
    }
}

// --- Обработка ответа от /chat API ---
function handleApiResponse(data) {
    hideError(); // Скрываем общую ошибку API, если была

    if (data.type === 'ai_message') {
        addMessageToChat('ai', data.ai_message);
    } else if (data.type === 'show_calculation_form') {
        addMessageToChat('ai', data.ai_message); // Показываем приглашение
        showEmbeddedForm(); // Показываем форму в interactive-area
    } else if (data.type === 'error') {
        addMessageToChat('ai', data.error_message || 'Произошла внутренняя ошибка.');
        showError(data.error_message || 'Произошла внутренняя ошибка.'); // Показываем и в чате и в секции ошибок
    } else {
        addMessageToChat('ai', 'Получен неожиданный ответ от сервера.');
    }

    // Показываем дисклеймер один раз после первого успешного ответа
    if (!disclaimerShown && (data.type === 'ai_message' || data.type === 'show_calculation_form')) {
        showDisclaimer("ВНИМАНИЕ! Этот инструмент предоставляет расчеты в ознакомительных целях и находится в стадии разработки. Данные могут быть неточными или не учитывать все детали вашей ситуации. Сервис не является официальной налоговой консультацией и не заменяет профессионального бухгалтера. Ответственность за правильность и своевременность уплаты налогов лежит на вас. Всегда сверяйте информацию с официальными источниками (Налоговый Кодекс РК, kgd.gov.kz) и/или консультируйтесь со специалистом."); // Замените на реальный текст
        disclaimerShown = false;
    }
}

// --- Работа с встраиваемой формой ---
function showEmbeddedForm() {
    removeEmbeddedForm(); // Удаляем старую форму на всякий случай
    const formNode = formTemplate.content.cloneNode(true);
    interactiveArea.appendChild(formNode);

    // Добавляем обработчики для новой формы
    const embeddedForm = document.getElementById('tax-form-embedded');
    const cancelBtn = document.getElementById('cancel-calc-form-btn');

    embeddedForm.addEventListener('submit', handleEmbeddedFormSubmit);
    cancelBtn.addEventListener('click', removeEmbeddedForm);
    chatInputArea.classList.add('hidden'); // Скрываем основное поле ввода чата, пока форма активна
}

function removeEmbeddedForm() {
    interactiveArea.innerHTML = ''; // Очищаем область
    chatInputArea.classList.remove('hidden'); // Показываем основное поле ввода чата
}

async function handleEmbeddedFormSubmit(event) {
    event.preventDefault();
    const form = event.target;
    const submitBtn = form.querySelector('#submit-calc-form-btn');
    const formErrorMessage = form.querySelector('#form-error-message');

    formErrorMessage.textContent = ''; // Сброс ошибки формы
    submitBtn.disabled = true;
    submitBtn.textContent = 'Расчет...';

    // Валидация на фронте (можно улучшить)
    const revenueInputEmbedded = form.querySelector('#revenue-embedded');
    const monthsWorkedInputEmbedded = form.querySelector('#months_worked-embedded');
    const revenue = parseFloat(revenueInputEmbedded.value);
    const monthsWorked = parseInt(monthsWorkedInputEmbedded.value, 10); // <<<--- Правильное имя переменной объявлено здесь

    if (isNaN(revenue) || revenue < 0) {
        formErrorMessage.textContent = 'Введите корректный доход.';
        submitBtn.disabled = false;
        submitBtn.textContent = 'Рассчитать';
        return;
    }
    if (isNaN(monthsWorked) || monthsWorked < 1 || monthsWorked > 6) {
        formErrorMessage.textContent = 'Введите корректное кол-во месяцев (1-6).';
        submitBtn.disabled = false;
        submitBtn.textContent = 'Рассчитать';
        return;
    }

    // Изменено здесь: ключ теперь 'months_worked'
    const requestData = {
        revenue: revenue,
        months_worked: monthsWorked // Ключ в JSON будет "months_worked"
    };

    try {
        const response = await fetch(CALC_API_URL, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(requestData),
        });

        if (!response.ok) {
            let errorDetails = `HTTP status: ${response.status}`;
            try {
                const errorData = await response.json();
                errorDetails += ` - ${errorData.error || errorData.message || JSON.stringify(errorData)}`;
            } catch(e) { /* ignore */ }
            throw new Error(`Ошибка сервера расчета: ${errorDetails}`);
        }

        const resultData = await response.json();
        removeEmbeddedForm(); // Убираем форму после успешной отправки
        displayCalculationResultInChat(resultData); // Отображаем результат в чате

    } catch (error) {
        console.error("Calculation API Error:", error);
        formErrorMessage.textContent = `Ошибка: ${error.message}`; // Показываем ошибку прямо в форме
        submitBtn.disabled = false;
        submitBtn.textContent = 'Рассчитать';
    }
}

// --- Отображение результатов расчета в чате ---
function displayCalculationResultInChat(data) {
    if (!data || !data.calculation || !data.explanation) {
        addMessageToChat('ai', 'Произошла ошибка при получении результатов расчета.');
        console.error("Invalid calculation result data:", data);
        return;
    }

    const resultNode = resultTemplate.content.cloneNode(true);
    const calc = data.calculation;
    const formatCurrency = (num) => num.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    const formatPercentage = (num) => num.toLocaleString('ru-RU', { minimumFractionDigits: 1, maximumFractionDigits: 1 });

    // Заполняем спаны с помощью data-result атрибутов
    resultNode.querySelector('[data-result="ipn"]').textContent = formatCurrency(calc.ipn);
    resultNode.querySelector('[data-result="sn"]').textContent = formatCurrency(calc.sn);
    resultNode.querySelector('[data-result="total-tax"]').textContent = formatCurrency(calc.total_tax);
    resultNode.querySelector('[data-result="opv"]').textContent = formatCurrency(calc.opv);
    resultNode.querySelector('[data-result="so"]').textContent = formatCurrency(calc.so);
    resultNode.querySelector('[data-result="vosms"]').textContent = formatCurrency(calc.vosms);
    resultNode.querySelector('[data-result="total-social"]').textContent = formatCurrency(calc.total_social);
    resultNode.querySelector('[data-result="limit-percentage"]').textContent = formatPercentage(calc.limit_percentage);

    // Варнинги
    const warningsContainer = resultNode.querySelector('[data-result="warnings"]');
    warningsContainer.innerHTML = ''; // Очищаем
    if (calc.warnings && calc.warnings.length > 0) {
        calc.warnings.forEach(warning => {
            const p = document.createElement('p');
            p.textContent = warning;
            warningsContainer.appendChild(p);
        });
    }

    // Объяснение и дисклеймер
    resultNode.querySelector('[data-result="explanation"]').textContent = data.explanation;
    resultNode.querySelector('[data-result="disclaimer"]').textContent = data.disclaimer;


    // Добавляем весь блок результата как одно "сообщение" AI
    const messageDiv = document.createElement('div');
    messageDiv.classList.add('chat-message', 'ai-message'); // Используем ai-message стиль
    messageDiv.appendChild(resultNode);
    chatContainer.appendChild(messageDiv);
    chatContainer.scrollTop = chatContainer.scrollHeight; // Прокрутка вниз

    // Показываем дисклеймер в подвале, если еще не показывали
    if (!disclaimerShown) {
        showDisclaimer(data.disclaimer);
        disclaimerShown = true;
    }
}


// --- Вспомогательные функции ---

function addMessageToChat(sender, text) {
    const messageDiv = document.createElement('div');
    messageDiv.classList.add('chat-message', sender === 'user' ? 'user-message' : 'ai-message');
    // Простая защита от вставки HTML
    const textNode = document.createTextNode(text);
    messageDiv.appendChild(textNode);

    // Добавляем в контейнер и прокручиваем
    chatContainer.appendChild(messageDiv);
    chatContainer.scrollTop = chatContainer.scrollHeight;
}

function setChatLoading(isLoading) {
    isWaitingForAi = isLoading;
    chatInput.disabled = isLoading;
    sendChatBtn.classList.toggle('hidden', isLoading);
    loadingChatBtn.classList.toggle('hidden', !isLoading);
}

function showError(message) {
    errorMessage.textContent = message;
    errorSection.classList.remove('hidden');
}
function hideError() {
    errorMessage.textContent = '';
    errorSection.classList.add('hidden');
}

function showDisclaimer(text) {
    disclaimerDiv.textContent = text || config.GetDisclaimer(); // На случай если текст не пришел
    disclaimerSection.classList.remove('hidden');
}