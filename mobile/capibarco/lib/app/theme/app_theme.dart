import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'app_colors.dart';

abstract final class AppTheme {
  static ThemeData light() {
    const colorScheme = ColorScheme(
      brightness: Brightness.light,
      primary: AppColors.primary,
      onPrimary: Colors.white,
      secondary: AppColors.secondary,
      onSecondary: Colors.white,
      error: Color(0xFFD94841),
      onError: Colors.white,
      surface: AppColors.background,
      onSurface: AppColors.text,
      surfaceContainerHighest: Color(0xFFF2E8E1),
      onSurfaceVariant: Color(0xFF61554F),
      outline: Color(0xFFD8C8BE),
      outlineVariant: Color(0xFFE9DDD6),
      shadow: Color(0x1A000000),
      scrim: Color(0x66000000),
      inverseSurface: AppColors.text,
      onInverseSurface: AppColors.background,
      inversePrimary: Color(0xFF9DE8D9),
      tertiary: AppColors.accent,
      onTertiary: AppColors.text,
      tertiaryContainer: Color(0xFFFFF1B4),
      onTertiaryContainer: AppColors.text,
      primaryContainer: Color(0xFFD8F6EF),
      onPrimaryContainer: AppColors.text,
      secondaryContainer: Color(0xFFFFD8CD),
      onSecondaryContainer: AppColors.text,
      errorContainer: Color(0xFFFFDAD6),
      onErrorContainer: Color(0xFF410002),
      surfaceDim: Color(0xFFEADFD7),
      surfaceBright: Color(0xFFFFF9F5),
      surfaceContainerLowest: AppColors.softCard,
      surfaceContainerLow: Color(0xFFFFF5F0),
      surfaceContainer: Color(0xFFF8EEE8),
      surfaceContainerHigh: Color(0xFFF4E9E3),
    );

    return _buildTheme(colorScheme);
  }

  static ThemeData dark() {
    const colorScheme = ColorScheme(
      brightness: Brightness.dark,
      primary: Color(0xFF74DCC8),
      onPrimary: Color(0xFF0D2F29),
      secondary: Color(0xFFFF9A80),
      onSecondary: Color(0xFF4D170A),
      error: Color(0xFFFFB4AB),
      onError: Color(0xFF690005),
      surface: AppColors.darkBackground,
      onSurface: AppColors.darkText,
      surfaceContainerHighest: Color(0xFF2A3236),
      onSurfaceVariant: Color(0xFFD1C3BC),
      outline: Color(0xFF8F8A85),
      outlineVariant: Color(0xFF474A4D),
      shadow: Color(0x66000000),
      scrim: Color(0x66000000),
      inverseSurface: AppColors.darkText,
      onInverseSurface: AppColors.darkBackground,
      inversePrimary: AppColors.primary,
      tertiary: Color(0xFFFFE08A),
      onTertiary: Color(0xFF3B2F00),
      tertiaryContainer: Color(0xFF574400),
      onTertiaryContainer: Color(0xFFFFF1B4),
      primaryContainer: Color(0xFF214C44),
      onPrimaryContainer: Color(0xFFD8F6EF),
      secondaryContainer: Color(0xFF6B2E1E),
      onSecondaryContainer: Color(0xFFFFD8CD),
      errorContainer: Color(0xFF93000A),
      onErrorContainer: Color(0xFFFFDAD6),
      surfaceDim: Color(0xFF121618),
      surfaceBright: Color(0xFF353D42),
      surfaceContainerLowest: Color(0xFF101416),
      surfaceContainerLow: Color(0xFF1B2124),
      surfaceContainer: AppColors.darkSurface,
      surfaceContainerHigh: Color(0xFF2A3134),
    );

    return _buildTheme(colorScheme);
  }

  static ThemeData _buildTheme(ColorScheme colorScheme) {
    final baseTextTheme = GoogleFonts.nunitoTextTheme().apply(
      bodyColor: colorScheme.onSurface,
      displayColor: colorScheme.onSurface,
    );
    const radius = 28.0;

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      scaffoldBackgroundColor: colorScheme.surface,
      textTheme: baseTextTheme,
      cardTheme: CardThemeData(
        color: colorScheme.surfaceContainerLowest,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radius),
        ),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: Colors.transparent,
        foregroundColor: colorScheme.onSurface,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: baseTextTheme.titleLarge?.copyWith(
          fontWeight: FontWeight.w800,
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colorScheme.surfaceContainerLow.withValues(alpha: 0.92),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radius),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radius),
          borderSide: BorderSide(color: colorScheme.outlineVariant),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radius),
          borderSide: BorderSide(color: colorScheme.primary, width: 1.5),
        ),
      ),
      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: colorScheme.surfaceContainerLowest.withValues(
          alpha: 0.92,
        ),
        height: 76,
        indicatorColor: colorScheme.primaryContainer,
        labelTextStyle: WidgetStatePropertyAll(
          baseTextTheme.labelMedium?.copyWith(fontWeight: FontWeight.w700),
        ),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: colorScheme.secondaryContainer.withValues(alpha: 0.7),
        side: BorderSide.none,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(999)),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size(0, 54),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(radius),
          ),
          textStyle: baseTextTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w800,
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(0, 54),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(radius),
          ),
        ),
      ),
    );
  }
}
