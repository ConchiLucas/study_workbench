from __future__ import annotations

import re
from dataclasses import dataclass

BLANK = "____"
WORD_TOKEN_RE = re.compile(r"[A-Za-z]+(?:[’'-][A-Za-z]+)*")
ANCHOR_RE = re.compile(r"[A-Za-z]+")
PLACEHOLDER_ANCHORS = {"sb", "sth", "somebody", "something"}

IRREGULAR_FORMS: dict[str, set[str]] = {
    "arise": {"arose", "arisen"},
    "be": {"am", "are", "is", "was", "were", "been", "being"},
    "bear": {"bore", "borne", "born"},
    "beat": {"beaten"},
    "become": {"became"},
    "begin": {"began", "begun"},
    "bend": {"bent"},
    "bestride": {"bestrode", "bestridden"},
    "bind": {"bound"},
    "bite": {"bit", "bitten"},
    "bleed": {"bled"},
    "blow": {"blew", "blown"},
    "break": {"broke", "broken"},
    "breed": {"bred"},
    "bring": {"brought"},
    "build": {"built"},
    "buy": {"bought"},
    "caveman": {"cavemen"},
    "catch": {"caught"},
    "choose": {"chose", "chosen"},
    "come": {"came"},
    "cost": {"cost"},
    "deal": {"dealt"},
    "dig": {"dug"},
    "do": {"did", "done"},
    "draw": {"drew", "drawn"},
    "drink": {"drank", "drunk"},
    "drive": {"drove", "driven"},
    "eat": {"ate", "eaten"},
    "fall": {"fell", "fallen"},
    "feed": {"fed"},
    "feel": {"felt"},
    "fight": {"fought"},
    "find": {"found"},
    "flee": {"fled"},
    "fly": {"flew", "flown"},
    "forbid": {"forbade", "forbidden"},
    "forget": {"forgot", "forgotten"},
    "forgive": {"forgave", "forgiven"},
    "freeze": {"froze", "frozen"},
    "get": {"got", "gotten"},
    "give": {"gave", "given"},
    "go": {"went", "gone"},
    "grow": {"grew", "grown"},
    "hang": {"hung"},
    "have": {"had"},
    "hear": {"heard"},
    "hide": {"hid", "hidden"},
    "hold": {"held"},
    "keep": {"kept"},
    "know": {"knew", "known"},
    "lay": {"laid"},
    "lead": {"led"},
    "leave": {"left"},
    "lend": {"lent"},
    "lie": {"lay", "lain"},
    "light": {"lit"},
    "lose": {"lost"},
    "make": {"made"},
    "mean": {"meant"},
    "meet": {"met"},
    "outshine": {"outshone"},
    "outgrow": {"outgrew", "outgrown"},
    "overhear": {"overheard"},
    "oversleep": {"overslept"},
    "pay": {"paid"},
    "read": {"read"},
    "ride": {"rode", "ridden"},
    "ring": {"rang", "rung"},
    "rise": {"rose", "risen"},
    "run": {"ran"},
    "say": {"said"},
    "see": {"saw", "seen"},
    "seek": {"sought"},
    "sell": {"sold"},
    "send": {"sent"},
    "shake": {"shook", "shaken"},
    "shoot": {"shot"},
    "show": {"shown"},
    "shrink": {"shrank", "shrunk"},
    "sing": {"sang", "sung"},
    "sink": {"sank", "sunk"},
    "sit": {"sat"},
    "sleep": {"slept"},
    "speak": {"spoke", "spoken"},
    "spend": {"spent"},
    "spin": {"spun"},
    "stand": {"stood"},
    "steal": {"stole", "stolen"},
    "stick": {"stuck"},
    "sting": {"stung"},
    "strike": {"struck", "stricken"},
    "swear": {"swore", "sworn"},
    "sweep": {"swept"},
    "swim": {"swam", "swum"},
    "take": {"took", "taken"},
    "teach": {"taught"},
    "tear": {"tore", "torn"},
    "tell": {"told"},
    "think": {"thought"},
    "throw": {"threw", "thrown"},
    "understand": {"understood"},
    "wake": {"woke", "woken"},
    "wear": {"wore", "worn"},
    "weep": {"wept"},
    "win": {"won"},
    "write": {"wrote", "written"},
    "child": {"children"},
    "foot": {"feet"},
    "goose": {"geese"},
    "man": {"men"},
    "mouse": {"mice"},
    "person": {"people"},
    "tooth": {"teeth"},
    "woman": {"women"},
}


@dataclass(frozen=True)
class BestSentenceCloze:
    cloze_sentence: str
    answer: str
    occurrences: int
    match_kind: str


def _literal_matches(text: str, target: str) -> list[re.Match[str]]:
    if not target:
        return []
    left = r"(?<![A-Za-z])" if target[0].isalpha() else ""
    right = r"(?![A-Za-z])" if target[-1].isalpha() else ""
    return list(re.finditer(left + re.escape(target) + right, text, re.IGNORECASE))


def _is_consonant(character: str) -> bool:
    return character.isalpha() and character not in "aeiou"


def _is_cvc(word: str) -> bool:
    return (
        len(word) >= 3
        and _is_consonant(word[-3])
        and word[-2] in "aeiou"
        and _is_consonant(word[-1])
        and word[-1] not in "wxy"
    )


def _inflected_forms(lemma: str) -> set[str]:
    lemma = lemma.lower()
    forms = {lemma, f"{lemma}'s", f"{lemma}’s"}
    forms.add(f"{lemma}ly")
    if lemma.endswith("y") and len(lemma) > 1 and _is_consonant(lemma[-2]):
        forms.add(f"{lemma[:-1]}ily")

    if lemma.endswith("y") and len(lemma) > 1 and _is_consonant(lemma[-2]):
        forms.update(
            {
                f"{lemma[:-1]}ies",
                f"{lemma[:-1]}ied",
                f"{lemma[:-1]}ier",
                f"{lemma[:-1]}iest",
            }
        )
    else:
        forms.add(f"{lemma}s")
        if lemma.endswith(("s", "x", "z", "ch", "sh", "o")):
            forms.add(f"{lemma}es")

    if lemma.endswith("e"):
        forms.add(f"{lemma}d")
    else:
        forms.add(f"{lemma}ed")

    if lemma.endswith("ie"):
        forms.add(f"{lemma[:-2]}ying")
    elif lemma.endswith("e") and not lemma.endswith(("ee", "oe", "ye")):
        forms.add(f"{lemma[:-1]}ing")
    else:
        forms.add(f"{lemma}ing")

    if lemma.endswith("e"):
        forms.update({f"{lemma}r", f"{lemma}st"})
    else:
        forms.update({f"{lemma}er", f"{lemma}est"})

    if _is_cvc(lemma):
        doubled = f"{lemma}{lemma[-1]}"
        forms.update({f"{doubled}ed", f"{doubled}ing", f"{doubled}er", f"{doubled}est"})

    if lemma.endswith("fe"):
        forms.add(f"{lemma[:-2]}ves")
    elif lemma.endswith("f"):
        forms.add(f"{lemma[:-1]}ves")
    if lemma.endswith("c"):
        forms.update({f"{lemma}ked", f"{lemma}king"})

    forms.update(IRREGULAR_FORMS.get(lemma, set()))
    return forms


def _build_result(
    sentence: str,
    matches: list[re.Match[str]],
    *,
    match_kind: str,
) -> BestSentenceCloze | None:
    if not matches:
        return None
    answers = {match.group(0).casefold() for match in matches}
    if len(answers) != 1:
        return None

    answer = matches[0].group(0)
    cloze_sentence = sentence
    for match in reversed(matches):
        cloze_sentence = f"{cloze_sentence[:match.start()]}{BLANK}{cloze_sentence[match.end():]}"
    if cloze_sentence.count(BLANK) != len(matches):
        return None
    if _literal_matches(cloze_sentence, answer):
        return None
    return BestSentenceCloze(
        cloze_sentence=cloze_sentence,
        answer=answer,
        occurrences=len(matches),
        match_kind=match_kind,
    )


def _primary_anchor(word: str) -> str:
    for part in ANCHOR_RE.findall(word):
        if part.casefold() not in PLACEHOLDER_ANCHORS:
            return part
    return ""


def build_best_sentence_cloze(word: str, sentence: str) -> BestSentenceCloze | None:
    target = word.strip()
    source = sentence.strip()
    if not target or not source or BLANK in source:
        return None

    exact_matches = _literal_matches(source, target)
    exact_result = _build_result(source, exact_matches, match_kind="exact")
    if exact_result is not None:
        return exact_result

    anchor = _primary_anchor(target)
    if not anchor:
        return None
    is_phrase_anchor = anchor.casefold() != target.casefold()

    anchor_matches = _literal_matches(source, anchor)
    anchor_result = _build_result(
        source,
        anchor_matches,
        match_kind="phrase_anchor" if is_phrase_anchor else "exact",
    )
    if anchor_result is not None:
        return anchor_result

    forms = _inflected_forms(anchor)
    token_matches = [
        match for match in WORD_TOKEN_RE.finditer(source) if match.group(0).casefold() in forms
    ]
    return _build_result(
        source,
        token_matches,
        match_kind="phrase_anchor" if is_phrase_anchor else "inflected",
    )
