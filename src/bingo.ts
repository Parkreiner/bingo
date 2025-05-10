/**
 * The minimum number of cards a player is allowed to have in a game.
 */
export const MIN_CARDS = 1;

/**
 * The maximum number of cards a player is allowed to have in a game.
 */
export const MAX_CARDS = 4;

/**
 * The number of cells that two cards are allowed to have in common to still be
 * called unique from a fun, gameplay standpoint. A unique cell, in this case,
 * refers to not just the numeric value of a cell, but also the position.
 *
 * The number is 2/3 of the total cells of a bingo card (25), excluding the free
 * space.
 */
const UNIQUENESS_THRESHOLD = 16;

export type BingoCell = Readonly<{
  value: number;
  daubed: boolean;
}>;

/**
 * A single, stateless representation of a bingo card.
 */
export type BingoCard = Readonly<{
  id: string;
  ownerId: string;

  /**
   * A 5x5 grid of Bingo cells. Each column corresponds to a different
   * "letter group" in the bingo board. That is:
   *
   * 1. Column 1 is column B and can have numbers 1–15
   * 2. Column 2 is column I and can have numbers 16–30
   * 3. Column 3 is column N and can have numbers 31–45, along with the free
   *    space in the middle
   * 4. Column 4 is column G and can have numbers 46–60
   * 5. Column 5 is column O and can have numbers 61–75
   *
   * The free space is represented as -1.
   */
  cells: readonly BingoCell[][];
}>;

function shuffleInPlace(input: unknown[]): void {
  for (let i = input.length - 1; i >= 1; i--) {
    const randomIndex = Math.floor(Math.random() * (i + 1));
    const elementToSwap = input[i];
    input[i] = input[randomIndex];
    input[randomIndex] = elementToSwap;
  }
}

function generateCellsForRange(start: number, end: number): number[] {
  const cells: number[] = [];
  for (let i = start; i <= end; i++) {
    cells.push(i);
  }
  shuffleInPlace(cells);

  return cells;
}

export function generateBingoCells(): number[][] {
  const allBCells = generateCellsForRange(1, 15);
  const allICells = generateCellsForRange(16, 30);
  const allNCells = generateCellsForRange(31, 45);
  const allGCells = generateCellsForRange(46, 60);
  const allOCells = generateCellsForRange(61, 75);

  const aggregateCells: number[][] = [
    allBCells.slice(0, 5),
    allICells.slice(0, 5),
    [...allNCells.slice(0, 2), -1, ...allNCells.slice(2, 4)],
    allGCells.slice(0, 5),
    allOCells.slice(0, 5),
  ];

  // Rotate the card so that it looks like a proper bingo card, and so that
  // fewer data transformations need to be done per render in the frontend
  for (let i = 0; i < aggregateCells.length; i++) {
    const row1 = aggregateCells[i];
    if (row1 === undefined) {
      throw new Error(`Went out of bounds for row 1 access at index ${i}`);
    }

    for (let j = i; j < row1.length; j++) {
      // This is really painful because of the noUnCheckedIndexedAccess compiler
      // setting
      const cell1 = row1[j];
      if (cell1 === undefined) {
        throw new Error(`Went out of bounds for cell 1 at [${i},${j}]`);
      }
      const row2 = aggregateCells[j];
      if (row2 === undefined) {
        throw new Error(`Went out of bounds for row 2 access at index ${j}`);
      }
      const cell2 = row2[i];
      if (cell2 === undefined) {
        throw new Error(`Went out of bounds for cell 2 at [${j},${i}]`);
      }

      row2[i] = cell1;
      row1[j] = cell2;
    }
  }

  return aggregateCells;
}

/**
 * @todo Need to make sure that no two users have the same card once this
 * logic gets moved to a server
 */
export function generateUniqueBingoCards(
  cardCount: number
): readonly BingoCard[] {
  const clamped = Math.max(MIN_CARDS, Math.min(MAX_CARDS, cardCount));
  if (!Number.isInteger(clamped)) {
    throw new Error(`Received invalid card count ${cardCount}`);
  }

  // Five layers of nesting in a loop isn't great, but the input elements are
  // guaranteed to be small enough that trying to move to a hashmap would
  // probably make the function perform worse
  const statelessCards: number[][][] = [];
  for (let i = 0; i < clamped; i++) {
    let cells: number[][];
    let cellConflicts: number;

    do {
      cells = generateBingoCells();
      cellConflicts = 0;

      for (const card of statelessCards) {
        for (const [i, row] of card.entries()) {
          for (const [j, cell] of row.entries()) {
            // Skip over the free space
            if (cell === -1) {
              continue;
            }

            const newCell = cells[i]?.[j];
            if (newCell === undefined) {
              throw new Error(`Went out of bounds at [${i},${j}]`);
            }
            if (cell === newCell) {
              cellConflicts++;
            }
          }
        }
      }
    } while (cellConflicts > UNIQUENESS_THRESHOLD);

    statelessCards.push(cells);
  }

  return statelessCards.map((card) => {
    return {
      id: window.isSecureContext
        ? window.crypto.randomUUID()
        : String(Math.random()).slice(2),

      ownerId: window.isSecureContext
        ? window.crypto.randomUUID()
        : String(Math.random()).slice(2),

      cells: card.map<BingoCell[]>((row) => {
        return row.map((cell) => {
          return {
            value: cell,
            daubed: false,
          };
        });
      }),
    };
  });
}
