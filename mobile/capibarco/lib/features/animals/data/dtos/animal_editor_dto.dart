import '../../domain/entities/animal_editor.dart';

class AnimalEditorDto {
  const AnimalEditorDto({
    required this.id,
    required this.name,
    required this.species,
    required this.breed,
    required this.sex,
    required this.size,
    required this.ageMonths,
    required this.description,
    required this.traits,
    required this.vaccinated,
    required this.sterilized,
    required this.city,
    required this.photoUrl,
    required this.status,
  });

  final String id;
  final String name;
  final String species;
  final String breed;
  final String sex;
  final String size;
  final int ageMonths;
  final String description;
  final List<String> traits;
  final bool vaccinated;
  final bool sterilized;
  final String city;
  final String photoUrl;
  final String status;

  factory AnimalEditorDto.fromJson(Map<String, dynamic> json) {
    final animal = json['animal'] as Map<String, dynamic>? ?? json;
    final photos = animal['photos'] as List<dynamic>? ?? const <dynamic>[];
    final firstPhoto = photos.isNotEmpty
        ? photos.first as Map<String, dynamic>
        : const <String, dynamic>{};
    final location =
        animal['location'] as Map<String, dynamic>? ??
        const <String, dynamic>{};

    return AnimalEditorDto(
      id: _stringValue(animal['animal_id']),
      name: _stringValue(animal['name']),
      species: _speciesValue(animal['species']),
      breed: _stringValue(animal['breed']),
      sex: _sexValue(animal['sex']),
      size: _sizeValue(animal['size']),
      ageMonths: _intValue(animal['age_months'], fallback: 0),
      description: _stringValue(animal['description']),
      traits: (animal['traits'] as List<dynamic>? ?? const <dynamic>[])
          .map(_stringValue)
          .where((item) => item.isNotEmpty)
          .toList(),
      vaccinated: _boolValue(animal['vaccinated']),
      sterilized: _boolValue(animal['sterilized']),
      city: _stringValue(location['city']),
      photoUrl: _stringValue(firstPhoto['url']),
      status: _statusValue(animal['status']),
    );
  }

  AnimalEditorEntity toDomain() {
    return AnimalEditorEntity(
      id: id,
      name: name,
      species: species,
      breed: breed,
      sex: sex,
      size: size,
      ageMonths: ageMonths,
      description: description,
      traits: traits,
      vaccinated: vaccinated,
      sterilized: sterilized,
      city: city,
      photoUrl: photoUrl,
      status: status,
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

int _intValue(Object? value, {int fallback = 0}) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  if (value is String) {
    return int.tryParse(value) ?? fallback;
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

String _sexValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'ANIMAL_SEX_MALE',
      2 => 'ANIMAL_SEX_FEMALE',
      3 => 'ANIMAL_SEX_UNKNOWN',
      _ => 'ANIMAL_SEX_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'ANIMAL_SEX_UNSPECIFIED');
}

String _sizeValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'ANIMAL_SIZE_SMALL',
      2 => 'ANIMAL_SIZE_MEDIUM',
      3 => 'ANIMAL_SIZE_LARGE',
      4 => 'ANIMAL_SIZE_EXTRA_LARGE',
      _ => 'ANIMAL_SIZE_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'ANIMAL_SIZE_UNSPECIFIED');
}

String _statusValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'ANIMAL_STATUS_DRAFT',
      2 => 'ANIMAL_STATUS_AVAILABLE',
      3 => 'ANIMAL_STATUS_RESERVED',
      4 => 'ANIMAL_STATUS_ADOPTED',
      5 => 'ANIMAL_STATUS_ARCHIVED',
      _ => 'ANIMAL_STATUS_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'ANIMAL_STATUS_UNSPECIFIED');
}
