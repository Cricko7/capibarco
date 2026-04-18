import '../../domain/entities/profile_animal_card.dart';

class ProfileAnimalCardDto {
  const ProfileAnimalCardDto({
    required this.id,
    required this.name,
    required this.species,
    required this.breed,
    required this.city,
    required this.photoUrl,
    required this.status,
  });

  final String id;
  final String name;
  final String species;
  final String breed;
  final String city;
  final String photoUrl;
  final String status;

  factory ProfileAnimalCardDto.fromJson(Map<String, dynamic> json) {
    final photos = json['photos'] as List<dynamic>? ?? const <dynamic>[];
    final location = json['location'] as Map<String, dynamic>? ?? const {};
    final firstPhoto = photos.isNotEmpty
        ? photos.first as Map<String, dynamic>
        : const <String, dynamic>{};

    return ProfileAnimalCardDto(
      id: json['animal_id'] as String? ?? '',
      name: json['name'] as String? ?? 'Pet',
      species: _enumLabel(json['species'], 'SPECIES'),
      breed: json['breed'] as String? ?? '',
      city: location['city'] as String? ?? '',
      photoUrl: firstPhoto['url'] as String? ?? '',
      status: _enumLabel(json['status'], 'ANIMAL_STATUS'),
    );
  }

  ProfileAnimalCardEntity toDomain() {
    return ProfileAnimalCardEntity(
      id: id,
      name: name,
      speciesLabel: species,
      breed: breed,
      city: city,
      photoUrl: photoUrl,
      statusLabel: status,
    );
  }

  static String _enumLabel(Object? raw, String prefix) {
    if (raw is num) {
      return raw.toString();
    }
    final value = raw as String? ?? '';
    if (value.isEmpty) {
      return '';
    }
    return value
        .replaceAll('${prefix}_', '')
        .replaceAll('_', ' ')
        .toLowerCase();
  }
}
