<script lang="ts">
	import { gsap } from "gsap";

	type Props = {
		targetSelector?: string;
		spinDuration?: number;
		hideDefaultCursor?: boolean;
		hoverDuration?: number;
		parallaxOn?: boolean;
	};

	let {
		targetSelector = ".cursor-target",
		spinDuration = 2,
		hideDefaultCursor = true,
		hoverDuration = 0.2,
		parallaxOn = true,
	}: Props = $props();

	let cursor = $state<HTMLDivElement | undefined>();
	let dot = $state<HTMLDivElement | undefined>();

	const isMobile = (() => {
		if (typeof window === "undefined") return false;
		const hasTouch = "ontouchstart" in window || navigator.maxTouchPoints > 0;
		const small = window.innerWidth <= 768;
		const ua = navigator.userAgent || navigator.vendor || "";
		const re = /android|webos|iphone|ipad|ipod|blackberry|iemobile|opera mini/i;
		return (hasTouch && small) || re.test(ua.toLowerCase());
	})();

	$effect(() => {
		if (isMobile || !cursor) return;
		const root = cursor;

		const corners = root.querySelectorAll<HTMLDivElement>(".target-cursor-corner");
		const constants = { borderWidth: 3, cornerSize: 12 };

		let activeTarget: Element | null = null;
		let currentLeaveHandler: (() => void) | null = null;
		let resumeTimeout: ReturnType<typeof setTimeout> | null = null;
		let targetCornerPositions: { x: number; y: number }[] | null = null;
		const activeStrength = { current: 0 };
		let tickerFn: (() => void) | null = null;
		let spinTl: gsap.core.Timeline | null = null;

		const originalCursor = document.body.style.cursor;
		if (hideDefaultCursor) document.body.style.cursor = "none";

		const cleanupTarget = (target: Element) => {
			if (currentLeaveHandler) target.removeEventListener("mouseleave", currentLeaveHandler);
			currentLeaveHandler = null;
		};

		gsap.set(root, {
			xPercent: -50,
			yPercent: -50,
			x: window.innerWidth / 2,
			y: window.innerHeight / 2,
		});

		const createSpin = () => {
			spinTl?.kill();
			spinTl = gsap
				.timeline({ repeat: -1 })
				.to(root, { rotation: "+=360", duration: spinDuration, ease: "none" });
		};
		createSpin();

		tickerFn = () => {
			if (!targetCornerPositions) return;
			const positions = targetCornerPositions;
			const strength = activeStrength.current;
			if (strength === 0) return;
			const cx = gsap.getProperty(root, "x") as number;
			const cy = gsap.getProperty(root, "y") as number;
			Array.from(corners).forEach((corner, i) => {
				const curX = gsap.getProperty(corner, "x") as number;
				const curY = gsap.getProperty(corner, "y") as number;
				const tx = positions[i].x - cx;
				const ty = positions[i].y - cy;
				const finalX = curX + (tx - curX) * strength;
				const finalY = curY + (ty - curY) * strength;
				const dur = strength >= 0.99 ? (parallaxOn ? 0.2 : 0) : 0.05;
				gsap.to(corner, {
					x: finalX,
					y: finalY,
					duration: dur,
					ease: dur === 0 ? "none" : "power1.out",
					overwrite: "auto",
				});
			});
		};

		const moveHandler = (e: MouseEvent) => {
			gsap.to(root, { x: e.clientX, y: e.clientY, duration: 0.1, ease: "power3.out" });
		};
		window.addEventListener("mousemove", moveHandler);

		const scrollHandler = () => {
			if (!activeTarget) return;
			const mx = gsap.getProperty(root, "x") as number;
			const my = gsap.getProperty(root, "y") as number;
			const under = document.elementFromPoint(mx, my);
			const still =
				under && (under === activeTarget || under.closest(targetSelector) === activeTarget);
			if (!still) currentLeaveHandler?.();
		};
		window.addEventListener("scroll", scrollHandler, { passive: true });

		const mouseDown = () => {
			if (!dot) return;
			gsap.to(dot, { scale: 0.7, duration: 0.3 });
			gsap.to(root, { scale: 0.9, duration: 0.2 });
		};
		const mouseUp = () => {
			if (!dot) return;
			gsap.to(dot, { scale: 1, duration: 0.3 });
			gsap.to(root, { scale: 1, duration: 0.2 });
		};
		window.addEventListener("mousedown", mouseDown);
		window.addEventListener("mouseup", mouseUp);

		const enterHandler = (ev: MouseEvent) => {
			const direct = ev.target as Element;
			let target: Element | null = null;
			let cur: Element | null = direct;
			while (cur && cur !== document.body) {
				if (cur.matches(targetSelector)) {
					target = cur;
					break;
				}
				cur = cur.parentElement;
			}
			if (!target) return;
			if (activeTarget === target) return;
			if (activeTarget) cleanupTarget(activeTarget);
			if (resumeTimeout) {
				clearTimeout(resumeTimeout);
				resumeTimeout = null;
			}
			activeTarget = target;
			Array.from(corners).forEach((corner) => gsap.killTweensOf(corner));
			gsap.killTweensOf(root, "rotation");
			spinTl?.pause();
			gsap.set(root, { rotation: 0 });

			const rect = target.getBoundingClientRect();
			const { borderWidth, cornerSize } = constants;
			const cx = gsap.getProperty(root, "x") as number;
			const cy = gsap.getProperty(root, "y") as number;

			targetCornerPositions = [
				{ x: rect.left - borderWidth, y: rect.top - borderWidth },
				{ x: rect.right + borderWidth - cornerSize, y: rect.top - borderWidth },
				{ x: rect.right + borderWidth - cornerSize, y: rect.bottom + borderWidth - cornerSize },
				{ x: rect.left - borderWidth, y: rect.bottom + borderWidth - cornerSize },
			];

			if (tickerFn) gsap.ticker.add(tickerFn);
			gsap.to(activeStrength, { current: 1, duration: hoverDuration, ease: "power2.out" });
			const positions = targetCornerPositions;

			Array.from(corners).forEach((corner, i) => {
				gsap.to(corner, {
					x: positions[i].x - cx,
					y: positions[i].y - cy,
					duration: 0.2,
					ease: "power2.out",
				});
			});

			const leaveHandler = () => {
				if (tickerFn) gsap.ticker.remove(tickerFn);
				targetCornerPositions = null;
				gsap.set(activeStrength, { current: 0, overwrite: true });
				activeTarget = null;
				const cs = Array.from(corners);
				gsap.killTweensOf(cs);
				const positions = [
					{ x: -cornerSize * 1.5, y: -cornerSize * 1.5 },
					{ x: cornerSize * 0.5, y: -cornerSize * 1.5 },
					{ x: cornerSize * 0.5, y: cornerSize * 0.5 },
					{ x: -cornerSize * 1.5, y: cornerSize * 0.5 },
				];
				const tl = gsap.timeline();
				cs.forEach((corner, i) => {
					tl.to(
						corner,
						{ x: positions[i].x, y: positions[i].y, duration: 0.3, ease: "power3.out" },
						0,
					);
				});

				resumeTimeout = setTimeout(() => {
					if (!activeTarget && spinTl) {
						const r = gsap.getProperty(root, "rotation") as number;
						const norm = r % 360;
						spinTl.kill();
						spinTl = gsap
							.timeline({ repeat: -1 })
							.to(root, { rotation: "+=360", duration: spinDuration, ease: "none" });
						gsap.to(root, {
							rotation: norm + 360,
							duration: spinDuration * (1 - norm / 360),
							ease: "none",
							onComplete: () => spinTl?.restart(),
						});
					}
					resumeTimeout = null;
				}, 50);
				cleanupTarget(target);
			};
			currentLeaveHandler = leaveHandler;
			target.addEventListener("mouseleave", leaveHandler);
		};

		window.addEventListener("mouseover", enterHandler);

		return () => {
			if (tickerFn) gsap.ticker.remove(tickerFn);
			window.removeEventListener("mousemove", moveHandler);
			window.removeEventListener("mouseover", enterHandler);
			window.removeEventListener("scroll", scrollHandler);
			window.removeEventListener("mousedown", mouseDown);
			window.removeEventListener("mouseup", mouseUp);
			if (activeTarget) cleanupTarget(activeTarget);
			spinTl?.kill();
			document.body.style.cursor = originalCursor;
		};
	});
</script>

{#if !isMobile}
	<div
		bind:this={cursor}
		class="pointer-events-none fixed top-0 left-0 z-[9999] h-0 w-0"
		style="will-change: transform;"
	>
		<div
			bind:this={dot}
			class="absolute top-1/2 left-1/2 h-1 w-1 -translate-x-1/2 -translate-y-1/2 rounded-full bg-white"
			style="will-change: transform;"
		></div>
		<div
			class="target-cursor-corner absolute top-1/2 left-1/2 h-3 w-3 -translate-x-[150%] -translate-y-[150%] border-[3px] border-r-0 border-b-0 border-white"
			style="will-change: transform;"
		></div>
		<div
			class="target-cursor-corner absolute top-1/2 left-1/2 h-3 w-3 translate-x-1/2 -translate-y-[150%] border-[3px] border-l-0 border-b-0 border-white"
			style="will-change: transform;"
		></div>
		<div
			class="target-cursor-corner absolute top-1/2 left-1/2 h-3 w-3 translate-x-1/2 translate-y-1/2 border-[3px] border-l-0 border-t-0 border-white"
			style="will-change: transform;"
		></div>
		<div
			class="target-cursor-corner absolute top-1/2 left-1/2 h-3 w-3 -translate-x-[150%] translate-y-1/2 border-[3px] border-r-0 border-t-0 border-white"
			style="will-change: transform;"
		></div>
	</div>
{/if}
