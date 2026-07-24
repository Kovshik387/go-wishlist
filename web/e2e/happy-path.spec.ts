import { expect, test } from "@playwright/test";

test("создание списка и желания вручную", async ({ page }) => {
  await page.goto("/");

  const onboarding = page.getByRole("dialog");
  if (await onboarding.isVisible().catch(() => false)) {
    await onboarding.getByRole("button", { name: "Дальше" }).click();
    await onboarding.getByRole("button", { name: "Дальше" }).click();
    await onboarding.getByRole("button", { name: "Начать" }).click();
  }

  await page.getByRole("link", { name: "Мои списки" }).click();
  await page.getByRole("link", { name: /Создать список/ }).first().click();
  await page.getByLabel("Название").fill("Подарки для поездки");
  await page.getByLabel(/Коротко о списке/).fill("Полезные вещи для следующего приключения");
  await page.getByRole("button", { name: "Создать список" }).click();

  await expect(page.getByRole("heading", { name: "Подарки для поездки" }).first()).toBeVisible();
  await page.getByRole("link", { name: /Добавить желание/ }).click();
  await page.getByRole("button", { name: "Вручную" }).click();
  await page.getByLabel("Название").fill("Термокружка");
  await page.getByLabel("Комментарий необязательно").fill("Небольшая, герметичная, синяя");
  await page.getByLabel("Цена").fill("2500");
  await page.getByRole("button", { name: "Добавить в список" }).click();

  await expect(page.getByRole("heading", { name: "Термокружка" })).toBeVisible();
});
