import { ref } from "vue";
import { driver } from "driver.js";
import "driver.js/dist/driver.css";
import { useAuthStore } from "@/stores/auth";
import { useUiStore } from "@/stores/ui";
import { computeOffsetForSprite } from "@/utils/spriteOffset";

export function useTutorial() {
  const isTutorialActive = ref(false);
  let pendingTimer: ReturnType<typeof setTimeout> | null = null;

  const steps = [
    {
      element: "#sprite-stone",
      popover: {
        title: "Soul Companion (灵魂伴侣)",
        description:
          "这块坚稳的石头是你倾诉心声的安全树洞。无论何时，点这里都可以开启一段充满共情的对话。",
        side: "top",
        align: "center",
      },
    },
    {
      element: "#sprite-bar",
      popover: {
        title: "Sisterhood Bond (姐妹纽带)",
        description:
          "去木屋看看大家在讨论什么吧！这里连接着广大的妈妈群体和资深认证的专家医疗人员。",
        side: "top",
        align: "start",
      },
    },
    {
      element: "#sprite-shell",
      popover: {
        title: "Echo / Memoir (记忆绘本)",
        description:
          "每一个贝壳都藏着一个关于你的专属回忆。提供几个标签，AI 就会为你生成充满怀旧感的个人绘本。",
        side: "top",
        align: "center",
      },
    },
    {
      element: "#sprite-car",
      popover: {
        title: "Partner Connection (伴侣视角)",
        description:
          "小车承载着生活的点滴。当你伴侣登录时，这里是你们情感互助交流的重要窗口。",
        side: "top",
        align: "end",
      },
    },
    {
      element: "#sprite-chair",
      popover: {
        title: "Tasks (伴侣任务)",
        description:
          "点这里查看当天的家庭任务，记录完成情况，或验收伴侣提交的任务。",
        side: "top",
        align: "center",
      },
    },
    {
      element: "#sprite-mailbox",
      popover: {
        title: "Whisper (海螺低语)",
        description:
          "把想说的话写进信箱，伴侣可以在这里读到你的悄悄话。",
        side: "top",
        align: "center",
      },
    },
    {
      element: "#sprite-crab",
      popover: {
        title: "小螃蟹 (随时向导)",
        description:
          "迷路了或者想探索隐藏功能？随时点这只小螃蟹，它会给你各种有趣的提示。",
        side: "right",
        align: "center",
      },
    },
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
