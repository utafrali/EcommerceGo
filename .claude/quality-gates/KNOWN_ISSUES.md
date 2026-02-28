# Known UI/UX Issues - Round 30

## FIXED IN ROUND 30 ✅
1. **CRITICAL**: Missing CSS animations (`animate-fade-in-slow`, `animate-underline-expand`) - **FIXED**
2. **HIGH**: Logo text inconsistency between Header and MobileDrawer - **FIXED**
3. **HIGH**: Missing Sign Out button in mobile drawer - **FIXED**
4. **HIGH**: Focus management missing in mobile drawer - **FIXED**

---

## REMAINING ISSUES (Medium/Low Priority)

### Medium Priority

#### 3. Z-Index Conflicts
**File**: `web/src/components/layout/Header.tsx:240`, `MegaMenu.tsx:40`, `SearchBar.tsx:184`
**Problem**: Header, MegaMenu, and SearchBar suggestions all use `z-50`. SearchBar suggestions might overlap with MegaMenu.
**Impact**: Desktop search suggestions could appear behind mega menu dropdowns.
**Fix**: Adjust z-index stack to: Header=50, SearchBar=51, MegaMenu=52, MobileDrawer=60

#### 4. Mobile Search UX Unclear
**File**: `web/src/components/layout/Header.tsx:304-324`
**Problem**: Search icon button opens entire mobile drawer instead of dedicated search interface.
**Impact**: Users expect dedicated search, not full menu.
**Fix**: Consider creating separate mobile search overlay or make it more obvious.

#### 6. No Visual Feedback for Mobile Drawer Trigger
**File**: `web/src/components/layout/Header.tsx:279-286`
**Problem**: Hamburger menu doesn't show active state when drawer is open.
**Impact**: No visual confirmation that menu is active.
**Fix**: Add conditional styling: `${isOpen ? 'bg-stone-100' : ''}`

#### 7. Mega Menu Close Behavior Inconsistency
**File**: `web/src/components/layout/Header.tsx:218-222`, `MegaMenu.tsx:41`
**Problem**: Header implements 200ms close delay, but MegaMenu's `onMouseLeave` bypasses it.
**Impact**: Confusing close timing.
**Fix**: Remove `onMouseLeave` from MegaMenu or increase Header delay.

#### 8. User Menu Dropdown Overflow
**File**: `web/src/components/layout/Header.tsx:355`
**Problem**: User menu (`w-52`) positioned `right-0` might overflow on md-lg breakpoints.
**Impact**: Dropdown could extend beyond viewport edge on tablets.
**Fix**: Add responsive width or use `right-0 md:right-auto` with position adjust.

#### 11. Cart Badge Accessibility
**File**: `web/src/components/layout/Header.tsx:415-419`, `MobileDrawer.tsx:254-258`
**Problem**: Cart count badge lacks screen reader context. "99+" read without knowing it's cart count.
**Impact**: Screen readers don't announce cart item count properly.
**Fix**: Add `<span className="sr-only">items in cart</span>` after badge.

---

### Low Priority

#### 5. Missing `aria-hidden` on Decorative Icons
**Files**: All icon components in Header and MobileDrawer
**Problem**: Inline SVG icons don't have `aria-hidden="true"` to mark them as decorative.
**Impact**: Screen readers might announce icons unnecessarily.
**Fix**: Add `aria-hidden="true"` to all decorative SVG elements.

#### 9. Category Expansion State Not Preserved
**File**: `web/src/components/layout/MobileDrawer.tsx:101`
**Problem**: Category expansion resets when drawer closes.
**Impact**: Poor UX if user reopens drawer.
**Fix**: Persist `expandedIds` in sessionStorage or parent component state.

#### 13. Multiple Hover Scale Transitions
**File**: `web/src/components/layout/Header.tsx:331,344,400,411`
**Problem**: `hover:scale-110` on multiple buttons can cause repaints.
**Impact**: Slight performance hit on mobile.
**Fix**: Replace with opacity/color transitions or use CSS `will-change`.

#### 14. Inline Styles in MegaMenu
**File**: `web/src/components/layout/MegaMenu.tsx:49-51`
**Problem**: Grid columns set via inline styles instead of Tailwind.
**Impact**: Prevents CSS purging, reduces consistency.
**Fix**: Generate Tailwind classes dynamically: `grid-cols-${columnCount}`.

#### 15. Silent Session Storage Errors
**File**: `web/src/components/layout/Header.tsx:148-156,201-208`
**Problem**: Storage errors silently ignored in try-catch.
**Impact**: Top bar dismissal won't persist in private browsing without feedback.
**Fix**: Add toast notification or console warning on storage failure.

---

## Master Agent Quality Control

To prevent similar issues in future:

1. **Run Quality Gates**: `.claude/quality-gates/pre-commit-checks.sh` before every commit
2. **Visual Checks**: Review http://localhost:3000 in browser after UI changes
3. **E2E Tests**: Run `npm run test:e2e` after significant changes
4. **Accessibility Audit**: Use Lighthouse or axe DevTools quarterly

---

## Next Steps

1. Fix Medium priority issues (estimated 2-3 hours)
2. Run comprehensive E2E tests
3. Perform Lighthouse audit
4. Document UI component patterns for consistency
