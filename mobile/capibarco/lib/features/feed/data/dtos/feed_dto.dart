import '../../domain/entities/feed_models.dart';

class FeedCardDto {
  const FeedCardDto({
    required this.id,
    required this.feedSessionId,
    required this.animalId,
    required this.ownerProfileId,
    required this.name,
    required this.species,
    required this.description,
    required this.ownerDisplayName,
    required this.photoUrl,
    required this.city,
    required this.boosted,
    required this.rankingReasons,
  });

  final String id;
  final String feedSessionId;
  final String animalId;
  final String ownerProfileId;
  final String name;
  final String species;
  final String description;
  final String ownerDisplayName;
  final String photoUrl;
  final String city;
  final bool boosted;
  final List<String> rankingReasons;

  factory FeedCardDto.fromJson(Map<String, dynamic> json) {
    final animal =
        json['animal'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    final photos = animal['photos'] as List<dynamic>? ?? const <dynamic>[];
    final firstPhoto = photos.isNotEmpty
        ? photos.first as Map<String, dynamic>
        : const <String, dynamic>{};
    final location =
        animal['location'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return FeedCardDto(
      id: _stringValue(json['feed_card_id']),
      feedSessionId: _stringValue(json['feed_session_id']),
      animalId: _stringValue(animal['animal_id']),
      ownerProfileId: _stringValue(animal['owner_profile_id']),
      name: _stringValue(animal['name'], fallback: 'Unknown'),
      species: _speciesValue(animal['species']),
      description: _stringValue(animal['description']),
      ownerDisplayName: _stringValue(
        json['owner_display_name'],
        fallback: 'Caregiver',
      ),
      photoUrl: _stringValue(firstPhoto['url']),
      city: _stringValue(location['city']),
      boosted: _boolValue(json['boosted']),
      rankingReasons:
          (json['ranking_reasons'] as List<dynamic>? ?? const <dynamic>[])
              .map((item) => _stringValue(item))
              .where((item) => item.isNotEmpty)
              .toList(),
    );
  }

  FeedCardEntity toDomain() {
    return FeedCardEntity(
      id: id,
      feedSessionId: feedSessionId,
      animalId: animalId,
      ownerProfileId: ownerProfileId,
      name: name,
      speciesLabel: species.replaceAll('SPECIES_', '').toLowerCase(),
      description: description,
      ownerDisplayName: ownerDisplayName,
      photoUrl: photoUrl,
      city: city,
      boosted: boosted,
      rankingReasons: rankingReasons,
    );
  }
}

String _stringValue(Object? value, {String fallback = ''}) {
  if (value == null) {
    return fallback;
  }
  if (value is String) {
    return value;
  }
  return value.toString();
}

bool _boolValue(Object? value, {bool fallback = false}) {
  if (value is bool) {
    return value;
  }
  if (value is num) {
    return value != 0;
  }
  if (value is String) {
    final normalized = value.trim().toLowerCase();
    if (normalized == 'true' || normalized == '1') {
      return true;
    }
    if (normalized == 'false' || normalized == '0') {
      return false;
    }
  }
  return fallback;
}

String _speciesValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'SPECIES_DOG',
      2 => 'SPECIES_CAT',
      3 => 'SPECIES_BIRD',
      4 => 'SPECIES_RABBIT',
      5 => 'SPECIES_RODENT',
      6 => 'SPECIES_REPTILE',
      7 => 'SPECIES_OTHER',
      _ => 'SPECIES_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'SPECIES_UNSPECIFIED');
}

class FeedPageDto {
  const FeedPageDto({
    required this.cards,
    required this.nextPageToken,
    required this.feedSessionId,
    this.isStale = false,
  });

  final List<FeedCardDto> cards;
  final String nextPageToken;
  final String feedSessionId;
  final bool isStale;

  factory FeedPageDto.fromJson(
    Map<String, dynamic> json, {
    bool isStale = false,
  }) {
    return FeedPageDto(
      cards: (json['cards'] as List<dynamic>? ?? const <dynamic>[])
          .map((item) => FeedCardDto.fromJson(item as Map<String, dynamic>))
          .toList(),
      nextPageToken: json['next_page_token'] as String? ?? '',
      feedSessionId: json['feed_session_id'] as String? ?? '',
      isStale: isStale,
    );
  }

  FeedPageEntity toDomain() {
    return FeedPageEntity(
      cards: cards.map((card) => card.toDomain()).toList(),
      nextPageToken: nextPageToken,
      feedSessionId: feedSessionId,
      isStale: isStale,
    );
  }
}
