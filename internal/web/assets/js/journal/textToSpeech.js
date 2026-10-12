import { setToWindow } from "../utils/index.js";

export function textToSpeech(button, inputQuery) {
	const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
	if (!SpeechRecognition) {
		console.warn("Speech recognition isn't supported in this browser.");
		button.disabled = true;
		button.title = "Voice dictation isn't supported in this browser";
		return;
	}
	const recognition = new SpeechRecognition();
	recognition.continuous = true;
	recognition.interimResults = false;

	// Only the user pressing the button (or a fatal error) ends dictation;
	// browsers end sessions on their own after pauses, so we restart those.
	let stopped = false;

	recognition.onresult = function(event) {
		const input = document.querySelector(inputQuery)
		for (let i = event.resultIndex; i < event.results.length; i++) {
			if (!event.results[i].isFinal) continue
			const transcript = event.results[i][0].transcript
			const needsSpace = input.value && !/\s$/.test(input.value) && !/^\s/.test(transcript)
			input.value = input.value + (needsSpace ? " " : "") + transcript
		}
	};

	recognition.onstart = function() {
		button.innerHTML = "<i class='fa fa-microphone-slash'></i>";
	};

	function resetButton() {
		button.innerHTML = "<i class='fa fa-microphone'></i>"
		button.onclick = function(e) {
			e.preventDefault()
			textToSpeech(button, inputQuery)
		}
	}

	recognition.onerror = function(event) {
		// Silence and brief network hiccups shouldn't end the session.
		if (event.error === "no-speech" || event.error === "aborted") return
		console.log(event)
		stopped = true
	};

	recognition.onend = function() {
		if (stopped) {
			resetButton()
			return
		}
		try {
			recognition.start()
		} catch (e) {
			resetButton()
		}
	};

	button.onclick = function(e) {
		e.preventDefault()
		stopped = true
		recognition.stop()
	};

	recognition.start();
}
setToWindow("textToSpeech", textToSpeech)
