#!/usr/bin/env node

// Broadcasts "Call for peace" message when package is installed in Russia, otherwise no-op

"use strict";

try {
	if (
		[
			"Asia/Anadyr", "Asia/Barnaul", "Asia/Chita", "Asia/Irkutsk", "Asia/Kamchatka",
			"Asia/Khandyga", "Asia/Krasnoyarsk", "Asia/Magadan", "Asia/Novokuznetsk",
			"Asia/Novosibirsk", "Asia/Omsk", "Asia/Sakhalin", "Asia/Srednekolymsk", "Asia/Tomsk",
			"Asia/Ust-Nera", "Asia/Vladivostok", "Asia/Yakutsk", "Asia/Yekaterinburg",
			"Europe/Astrakhan", "Europe/Kaliningrad", "Europe/Kirov", "Europe/Moscow",
			"Europe/Samara", "Europe/Saratov", "Europe/Simferopol", "Europe/Ulyanovsk",
			"Europe/Volgograd", "W-SU"
		].indexOf(new Intl.DateTimeFormat().resolvedOptions().timeZone) === -1
	) {
		return;
	}

	var bold = function (msg) { return "\x1b[97m\x1b[1m" + msg + "\x1b[22m\x1b[37m"; };

	var flags = function () {
		return (
			"🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦" +
			"  🇷🇺  🇺🇦  🇷🇺  🇺🇦  🇷🇺  🇺🇦"
		);
	};

	process._rawDebug(
		[
			"", flags(), "",
			"\x1b[37m24 февраля " +
				bold("Россия напала на Украину") +
				", что уже понесло за собой " +
				bold("3500-6000 жертв среди русских военных") +
				" и примерно " +
				bold("2000-4000 - среди украинских") +
				", а также " +
				bold("549 убитых граждан, в том числе 26 детей") +
				".",
			"",
			"Народ Украины полностью мобилизован и готов защищать свою страну от вражеского" +
				" вторжения. " +
				bold(
					"91% украинцев полностью поддерживает своего президента Владимира Зеленского"
				) +
				" и его ответные действия на нападение России.",
			"",
			"Весь мир осудил необоснованное вторжение и решил ввести " +
				bold("невиданные ранее санкции против России") +
				". С каждым новым днем они будут ощущаться все сильнее и сильнее среди мирных" +
				" граждан.",
			"",
			"В то же время, " +
				bold("российское правительство ограничивает доступ граждан ко внешней информации") +
				", насаживая единоправильные формулировки и версии происходящего.",
			"", "В качестве надежного источника информации загрузите безопасный браузер Tor:", "",
			bold("https://www.torproject.org/"), "", "И посетите:", "",
			bold(
				"https://www.bbcweb3hytmzhn5d532owbu6oqadra5z3ar726vq5kgwwn6aucdccrad.onion/russian"
			),
			"", flags(), "",
			bold("Прекратите эту бессмысленную войну! Остановите Путина!") + "\x1b[39m", "",
			flags(), ""
		].join("\n")
	);
} catch (error) {
	// ignore
}
