/**
 * WHAT "SAVE" MEANS: THE SHOP'S WORD, NOT THE BROWSER'S.
 *
 * `saveRow`, `deleteRow` and `newId` keep the names every management screen
 * already calls, and now go to the shop (src/lib/manage-shoes-lines-categories-
 * reviews-and-orders.ts). They return promises that never reject: an outcome
 * says what the shop did — the record it made or changed, the fields it
 * refused, its own sentence, "the session ended", or "unreachable".
 *
 * NOTHING IS WRITTEN HERE ON THE SHOP'S BEHALF. A screen that has saved
 * draws what the shop answered or reads the list again; it never redraws
 * from what was typed, so a name the shop trimmed or an id it made shows as
 * stored. `setCollection` is untouched and still belongs to the Studio.
 */
import type { Row } from "../../data";
import {
  createRecord, removeRecord, setPairs, updateRecord, uploadPicture, type Body, type Outcome,
} from "../../lib/manage-shoes-lines-categories-reviews-and-orders";

export type { Failure, Outcome, SizeAnswer } from "../../lib/manage-shoes-lines-categories-reviews-and-orders";

/**
 * A key for a thing the browser draws before the shop has answered.
 *
 * A COUNTER AND NOT THE LIST'S LENGTH, so two made inside one millisecond
 * never repeat. It is never sent to the shop as a record's id: the shop makes
 * that, and the list shows the one it made.
 */
let made = 0;

export function newId(collection: string): string {
  made += 1;
  return `${collection}-${Date.now().toString(36)}-${made}`;
}

/**
 * Creates a record, or changes the one whose id the row carries. The id is the
 * address of a change and is never part of what is sent.
 */
export function saveRow(collection: string, row: Body & { id?: string }): Promise<Outcome<Row>> {
  const { id, ...body } = row;
  return id ? updateRecord(collection, id, body) : createRecord(collection, body);
}

export function deleteRow(collection: string, id: string): Promise<Outcome<null>> {
  return removeRecord(collection, id);
}

/** A picture to the shop; the outcome's value is the address to keep on the record. */
export const uploadFile = uploadPicture;

/** One size's count on its own, with the count the person last saw. */
export const saveCount = setPairs;
