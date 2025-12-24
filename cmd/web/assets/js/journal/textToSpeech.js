import { setToWindow } from "../utils/index.js";

export function textToSpeech(button, inputQuery) {
	const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
	const recognition = new SpeechRecognition();
	recognition.continuous = true;

	recognition.onresult = function(event) {
		const current = event.resultIndex;
		const transcript = event.results[current][0].transcript;
		const input = document.querySelector(inputQuery)
		input.value = input.value + transcript
	};

	recognition.onstart = function() {
		button.innerHTML = "<i class='fa fa-microphone-slash'></i>";
		console.log('Voice recognition activated. Try speaking.');
	};

	function resetButton() {
		button.innerHTML = "<i class='fa fa-microphone'></i>"
		button.onclick = function(e) {
			e.preventDefault()
			textToSpeech(button, inputQuery)
		}
	}

	recognition.onspeechend = function() {
		console.log('You have stopped speaking.');
		resetButton()
	};

	recognition.onerror = function(event) {
		console.log(event)
		resetButton()
	};

	button.onclick = function(e) {
		e.preventDefault()
		recognition.stop()
	};

	recognition.start();
}
setToWindow("textToSpeech", textToSpeech)
