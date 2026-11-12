import { ref } from "vue";
import { driver } from "driver.js";
import "driver.js/dist/driver.css";
import { useAuthStore } from "@/stores/auth";
import { useUiStore } from "@/stores/ui";
import { computeOffsetForSprite } from "@/utils/spriteOffset";

export function useTutorial() {
  const isTutorialActive = ref(false);
  let pendingTimer: ReturnType<typeof setTimeout> | null = null;

  function step(id: string, title: string, description: string, align: string, side = "top") {
    return { element: `#sprite-${id}`, popover: { title, description, side, align } };
  }

  const steps = [
    step("stone", "Soul Companion (灵魂伴侣)", "这块坚稳的石头是你倾诉心声的安全树洞。无论何时，点这里都可以开启一段充满共情的对话。", "center"),
    step("bar", "Sisterhood Bond (姐妹纽带)", "去木屋看看大家在讨论什么吧！这里连接着广大的妈妈群体和资深认证的专家医疗人员。", "start"),
    step("car", "Partner Connection (伴侣视角)", "小车承载着生活的点滴。当你伴侣登录时，这里是你们情感互助交流的重要窗口。", "end"),
    step("chair", "Tasks (任务回执)", "这边会同步他提交的任务回执。完成、验收、留存纪念卡，都会从这里串起来。", "center"),
    step("mailbox", "Whisper Intel (心语信箱)", "信箱会把你的问卷选择和一句心愿整理成可执行的心语情报，再发给伴侣。", "center"),
    step("crab", "小螃蟹 (随时向导)", "迷路了或者想探索隐藏功能？随时点这只小螃蟹，它会给你各种有趣的提示。", "center", "right"),
  ];

  /** Scroll the parallax view to center the sprite for the given step,
   *  then call the callback after a short delay to let the animation finish. */
  function scrollToStep(stepIndex: number, callback: () => void) {
    const step = steps[stepIndex];
    if (!step) {
      callback();
      return;
    }
    const spriteId = step.element.replace("#sprite-", "");
    const offset = computeOffsetForSprite(spriteId);
    const uiStore = useUiStore();
    if (offset === null) {
      callback();
    } else {
      uiStore.parallaxScrollTo(offset);
      // Wait for parallax animation to settle before driver.js highlights
      setTimeout(callback, 450);
    }
  }

  let currentStepIndex = 0;

  const driverObj = driver({
    showProgress: true,
    allowClose: true,
    overlayColor: "rgba(0, 0, 0, 0.5)",
    nextBtnText: "下一步",
    prevBtnText: "上一步",
    doneBtnText: "完成",
    progressText: "{{current}} / {{total}}",
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    steps: steps as any,
    animate: false, // We handle scrolling ourselves
    onDestroyStarted: () => {
      const authStore = useAuthStore();
      if (!driverObj.hasNextStep() || confirm("确定要退出新手指引吗？")) {
        driverObj.destroy();
        isTutorialActive.value = false;
        authStore.completeTutorial();
      }
    },
    onNextClick: () => {
      currentStepIndex++;
      scrollToStep(currentStepIndex, () => {
        driverObj.moveNext();
      });
    },
    onPrevClick: () => {
      currentStepIndex--;
      scrollToStep(currentStepIndex, () => {
        driverObj.movePrevious();
      });
    },
  });

  function startTutorial() {
    const authStore = useAuthStore();
    const uiStore = useUiStore();

    // Check account-bound state: skip if user already completed the tutorial
    if (authStore.user?.tutorial_completed) return;

    // Guest users always see the tutorial (no persistence)

    // Cancel any previously pending tutorial launch
    cancelPending();

    // Don't start if a panel is already open
    if (uiStore.activePanel) return;

    currentStepIndex = 0;

    // Scroll to first sprite, then start the tutorial
    pendingTimer = setTimeout(() => {
      pendingTimer = null;
      // Re-check: if a panel opened during the delay, abort
      if (uiStore.activePanel) return;
      scrollToStep(0, () => {
        if (uiStore.activePanel) return;
        isTutorialActive.value = true;
        driverObj.drive();
      });
    }, 500);
  }

  /** Cancel a pending tutorial start (e.g. if a panel was opened) */
  function cancelPending() {
    if (pendingTimer) {
      clearTimeout(pendingTimer);
      pendingTimer = null;
    }
  }

  /** Stop tutorial if it is currently active */
  function stopTutorial() {
    cancelPending();
    if (isTutorialActive.value) {
      driverObj.destroy();
      isTutorialActive.value = false;
    }
  }

  function forceStartTutorial() {
    currentStepIndex = 0;
    scrollToStep(0, () => {
      isTutorialActive.value = true;
      driverObj.drive();
    });
  }

  return {
    isTutorialActive,
    startTutorial,
    stopTutorial,
    cancelPending,
    forceStartTutorial,
  };
}
